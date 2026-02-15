package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

func BackupArtifactsFeeds(orgURL, project, backupPath, resourceGUID string) error {
	feeds, err := ListFeeds(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, f := range feeds {
		fp := filepath.Join(backupPath, safeFeedFile(f.Name))
		b, _ := json.MarshalIndent(f, "", "  ")
		if err := os.WriteFile(fp, b, 0644); err != nil {
			return err
		}
		fmt.Println("✔ Backed up feed:", f.Name)
	}

	return nil
}

func safeFeedFile(name string) string {
	s := strings.TrimSpace(name)
	re := regexp.MustCompile(`[^a-zA-Z0-9._-]+`)
	s = re.ReplaceAllString(s, "_")
	if s == "" {
		s = "feed"
	}
	return s + ".json"
}
