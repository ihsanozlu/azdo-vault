package internal

import (
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

func RemapReleaseArtifacts(
	payload map[string]any,
	targetOrgURL, targetProject, resourceGUID string,
) error {

	arts, ok := payload["artifacts"].([]any)
	if !ok || len(arts) == 0 {
		return nil
	}

	tgtProj, err := GetProjectInfo(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return fmt.Errorf("get target project info failed: %w", err)
	}
	tgtProjectID := tgtProj.Id
	tgtProjectName := tgtProj.Name

	repos, err := ListRepos(targetOrgURL, targetProject)
	if err != nil {
		return fmt.Errorf("list target repos failed: %w", err)
	}
	repoIDByName := map[string]string{}
	for _, r := range repos {
		if r.Id != "" && r.Name != "" {
			repoIDByName[strings.ToLower(r.Name)] = r.Id
		}
	}

	// target build defs map (name -> id)
	bdefs, err := ListBuildDefinitions(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return fmt.Errorf("list target build definitions failed: %w", err)
	}
	buildIDByName := map[string]int{}
	for _, d := range bdefs {
		if d.Id != 0 && d.Name != "" {
			buildIDByName[d.Name] = d.Id
		}
	}

	for _, aAny := range arts {
		a, _ := aAny.(map[string]any)
		if a == nil {
			continue
		}

		typ, _ := a["type"].(string)

		defRef, _ := a["definitionReference"].(map[string]any)
		if defRef == nil {
			continue
		}

		if proj, ok := defRef["project"].(map[string]any); ok && proj != nil {
			proj["id"] = tgtProjectID
			proj["name"] = tgtProjectName
		}

		switch strings.ToLower(typ) {
		case "git":
			// In Git artifacts, definition = repo
			defObj, _ := defRef["definition"].(map[string]any)
			if defObj == nil {
				continue
			}

			repoName, _ := defObj["name"].(string)
			if strings.TrimSpace(repoName) == "" {
				return fmt.Errorf("git artifact has empty repo name")
			}

			repoID := repoIDByName[strings.ToLower(repoName)]
			if repoID == "" {
				return fmt.Errorf("target repo not found for git artifact repo '%s'", repoName)
			}

			defObj["id"] = repoID

			// sourceId: "{projectId}:{repoId}"
			a["sourceId"] = fmt.Sprintf("%s:%s", tgtProjectID, repoID)

		case "build":
			// In Build artifacts, definition = build definition
			defObj, _ := defRef["definition"].(map[string]any)
			if defObj == nil {
				continue
			}

			buildName, _ := defObj["name"].(string)
			if strings.TrimSpace(buildName) == "" {
				return fmt.Errorf("build artifact has empty build definition name")
			}

			newBuildID := buildIDByName[buildName]
			if newBuildID == 0 {
				return fmt.Errorf("target build definition not found for '%s'", buildName)
			}

			defObj["id"] = strconv.Itoa(newBuildID)

			// sourceId: "{projectId}:{buildDefId}"
			a["sourceId"] = fmt.Sprintf("%s:%d", tgtProjectID, newBuildID)

			if uObj, ok := defRef["artifactSourceDefinitionUrl"].(map[string]any); ok && uObj != nil {
				if raw, ok := uObj["id"].(string); ok && raw != "" {
					uObj["id"] = rewriteArtifactSourceURL(raw, tgtProjectID, newBuildID)
				}
			}
		default:
			// leave unknown artifact types unchanged, but project reference already updated
		}
	}

	return nil
}

func rewriteArtifactSourceURL(raw string, targetProjectID string, targetBuildDefID int) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	q := u.Query()
	if targetProjectID != "" {
		q.Set("projectId", targetProjectID)
	}
	if targetBuildDefID != 0 {
		q.Set("definitionId", strconv.Itoa(targetBuildDefID))
	}
	u.RawQuery = q.Encode()
	return u.String()
}
