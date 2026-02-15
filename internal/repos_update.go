package internal

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// UpdateRepoDefaultBranch updates repository.defaultBranch using Azure DevOps REST.
// PATCH /_apis/git/repositories/{repositoryId}
func UpdateRepoDefaultBranch(orgURL, project, repoID, defaultBranch, resourceGUID string) error {
	if strings.TrimSpace(repoID) == "" {
		return fmt.Errorf("repoID is empty")
	}
	if strings.TrimSpace(defaultBranch) == "" {
		return fmt.Errorf("defaultBranch is empty")
	}
	if !strings.HasPrefix(defaultBranch, "refs/heads/") {
		defaultBranch = "refs/heads/" + strings.TrimPrefix(defaultBranch, "/")
	}

	uri := fmt.Sprintf("%s/%s/_apis/git/repositories/%s?api-version=7.2-preview.2",
		strings.TrimRight(orgURL, "/"),
		project,
		repoID,
	)

	body := map[string]any{
		"defaultBranch": defaultBranch,
	}
	bodyBytes, _ := json.Marshal(body)

	args := []string{
		"rest",
		"--method", "patch",
		"--uri", uri,
		"--headers", "Content-Type=application/json",
		"--body", string(bodyBytes),
		"--only-show-errors",
		"--output", "none",
	}
	if strings.TrimSpace(resourceGUID) != "" {
		args = append(args, "--resource", resourceGUID)
	}

	cmd := exec.Command("az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("az rest update repo default branch failed: %w\n%s", err, string(out))
	}

	return nil
}
