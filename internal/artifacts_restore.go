package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func RestoreArtifactsFeedsFromBackup(sourceOrgURL, sourceProject, targetOrgURL, targetProject, backupPath string, selected []string, resourceGUID string) error {
	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}
	restoreAll := len(selected) == 1 && strings.EqualFold(selected[0], "all")

	targetFeeds, err := ListFeeds(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return err
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

		var feed Feed
		if err := json.Unmarshal(b, &feed); err != nil {
			return err
		}

		if existing := FindFeedByName(targetFeeds, feed.Name); existing != nil {
			fmt.Println("✔ Feed exists, skipping:", feed.Name)
			continue
		}

		// Create payload (only fields ADO accepts for create)
		payload := map[string]any{
			"name":            feed.Name,
			"upstreamEnabled": feed.UpstreamEnabled,
		}
		if len(feed.UpstreamSources) > 0 {
			// some orgs accept upstreamSources on create; if not, we add a separate "update" later.
			payload["upstreamSources"] = feed.UpstreamSources
		}

		fmt.Println("Creating feed:", feed.Name)
		created, err := CreateFeed(targetOrgURL, targetProject, resourceGUID, payload)
		if err != nil {
			fmt.Printf("⚠ Failed creating feed '%s': %v\n", feed.Name, err)
			continue
		}
		fmt.Println("✔ Created feed:", created.Name)
	}

	fmt.Println("✔ Artifacts feeds restore finished")
	return nil
}
