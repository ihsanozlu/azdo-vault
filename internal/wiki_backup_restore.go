package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func BackupWikis(orgURL, project, backupPath string, selected []string, resourceGUID string) error {
	wikis, err := ListWikis(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	backupAll := len(selected) == 1 && strings.EqualFold(selected[0], "all")

	// repos map so we can locate wiki repo remoteUrl for ProjectWiki
	repos, err := ListRepos(orgURL, project)
	if err != nil {
		return fmt.Errorf("backup wikis: list repos failed: %w", err)
	}
	repoByID := map[string]Repo{}
	for _, r := range repos {
		repoByID[strings.ToLower(r.Id)] = r
	}

	for _, w := range wikis {
		if !backupAll && !contains(selected, w.Name) && !contains(selected, w.ID) {
			continue
		}

		// write wiki metadata json
		fn := fmt.Sprintf("%s_%s.json", safeFilePart(w.ID), safeFilePart(w.Name))
		fp := filepath.Join(backupPath, fn)

		b, err := json.MarshalIndent(w, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(fp, b, 0644); err != nil {
			return err
		}
		fmt.Println("✔ Backed up wiki metadata:", w.Name, "type=", w.Type)

		// If ProjectWiki => mirror clone its backing repo
		if IsProjectWiki(w) && strings.TrimSpace(w.RepositoryID) != "" {
			r, err := GetRepoByID(orgURL, project, w.RepositoryID, resourceGUID)
			if err != nil || strings.TrimSpace(r.RemoteURL) == "" {
				fmt.Printf("⚠ wiki '%s': repo remoteUrl not found via REST for repositoryId=%s: %v\n",
					w.Name, w.RepositoryID, err)
				continue
			}

			destRepoDir := filepath.Join(backupPath, safeFilePart(w.Name)+".wiki.git")
			if err := MirrorClone(r.RemoteURL, destRepoDir); err != nil {
				fmt.Printf("⚠ wiki '%s': git mirror clone failed: %v\n", w.Name, err)
				continue
			}
			fmt.Println("✔ Backed up wiki repo:", w.Name, "->", destRepoDir)
		}
	}

	fmt.Println("✔ Wikis backup finished")
	return nil
}

func RestoreWikisFromBackup(
	sourceOrgURL, sourceProject string,
	targetOrgURL, targetProject, backupPath string,
	selected []string,
	resourceGUID string,
) error {
	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}
	restoreAll := len(selected) == 1 && strings.EqualFold(selected[0], "all")

	targetWikis, err := ListWikis(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return err
	}

	// repos for mapping CodeWiki repositoryId (source repo -> name -> target repoId)
	sourceRepos, err := ListRepos(sourceOrgURL, sourceProject)
	if err != nil {
		return fmt.Errorf("restore wikis: list source repos failed: %w", err)
	}
	sourceRepoNameByID := map[string]string{}
	for _, r := range sourceRepos {
		sourceRepoNameByID[strings.ToLower(r.Id)] = r.Name
	}

	targetRepos, err := ListRepos(targetOrgURL, targetProject)
	if err != nil {
		return fmt.Errorf("restore wikis: list target repos failed: %w", err)
	}
	targetRepoIDByName := map[string]string{}
	for _, r := range targetRepos {
		targetRepoIDByName[strings.ToLower(r.Name)] = r.Id
	}
	targetRepoByID := map[string]Repo{}
	for _, r := range targetRepos {
		targetRepoByID[strings.ToLower(r.Id)] = r
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}
		if !restoreAll && !contains(selected, f.Name()) && !contains(selected, strings.TrimSuffix(f.Name(), ".json")) {
			continue
		}

		fp := filepath.Join(backupPath, f.Name())
		b, err := os.ReadFile(fp)
		if err != nil {
			return err
		}

		var w Wiki
		if err := json.Unmarshal(b, &w); err != nil {
			return err
		}

		if FindWikiByName(targetWikis, w.Name) != nil {
			fmt.Println("✔ Wiki exists, skipping:", w.Name)
			continue
		}

		proj, err := GetProjectInfo(targetOrgURL, targetProject, resourceGUID)
		if err != nil {
			return err
		}
		targetProjectID := proj.Id

		// Create wiki definition in target
		payload := map[string]any{
			"name":      w.Name,
			"type":      w.Type,
			"projectId": targetProjectID,
		}

		if IsCodeWiki(w) {
			// Map repo by name from source repoId -> source repoName -> target repoId
			srcRepoName := sourceRepoNameByID[strings.ToLower(w.RepositoryID)]
			if strings.TrimSpace(srcRepoName) == "" {
				fmt.Printf("⚠ CodeWiki '%s': source repositoryId not found in source project; skipping\n", w.Name)
				continue
			}
			targetRepoID := targetRepoIDByName[strings.ToLower(srcRepoName)]
			if strings.TrimSpace(targetRepoID) == "" {
				fmt.Printf("⚠ CodeWiki '%s': target repo '%s' not found; skipping\n", w.Name, srcRepoName)
				continue
			}

			payload["repositoryId"] = targetRepoID
			if strings.TrimSpace(w.MappedPath) != "" {
				payload["mappedPath"] = w.MappedPath
			} else {
				// common default for code wiki if missing
				payload["mappedPath"] = "/"
			}
		}

		fmt.Println("Creating wiki:", w.Name, "type=", w.Type)
		created, err := CreateWiki(targetOrgURL, targetProject, resourceGUID, payload)
		if err != nil {
			fmt.Printf("⚠ Failed creating wiki '%s': %v\n", w.Name, err)
			continue
		}
		fmt.Println("✔ Created wiki:", created.Name)

		// If ProjectWiki: push mirrored repo content into created.RepositoryID
		if IsProjectWiki(w) {
			srcMirrorDir := filepath.Join(backupPath, safeFilePart(w.Name)+".wiki.git")
			if _, err := os.Stat(srcMirrorDir); err != nil {
				fmt.Printf("⚠ ProjectWiki '%s': mirror repo dir not found (%s); skipping git push\n", w.Name, srcMirrorDir)
				continue
			}

			if strings.TrimSpace(created.RepositoryID) == "" {
				fmt.Printf("⚠ ProjectWiki '%s': created wiki missing repositoryId; cannot push\n", w.Name)
				continue
			}

			tr, err := GetRepoByID(targetOrgURL, targetProject, created.RepositoryID, resourceGUID)
			if err != nil || strings.TrimSpace(tr.RemoteURL) == "" {
				fmt.Printf("⚠ ProjectWiki '%s': target wiki repo remoteUrl not found via REST for repositoryId=%s: %v\n",
					w.Name, created.RepositoryID, err)
				continue
			}

			if err := MirrorPush(srcMirrorDir, tr.RemoteURL); err != nil {
				fmt.Printf("⚠ ProjectWiki '%s': git mirror push failed: %v\n", w.Name, err)
				continue
			}
			fmt.Println("✔ Pushed wiki repo:", w.Name)

		}
	}

	fmt.Println("✔ Wikis restore finished")
	return nil
}
