package internal

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Repo struct {
	Id         string `json:"id"`
	Name       string `json:"name"`
	RemoteURL  string `json:"remoteUrl"`
	IsDisabled bool   `json:"isDisabled"`
}

func ListRepos(orgURL, project string) ([]Repo, error) {
	cmd := exec.Command("az", "repos", "list", "--organization", orgURL, "--project", project, "--output", "json")
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var repos []Repo
	if err := json.Unmarshal(out, &repos); err != nil {
		return nil, err
	}

	var enabled []Repo
	for _, r := range repos {
		if !r.IsDisabled {
			enabled = append(enabled, r)
		}
	}

	return enabled, nil
}

type AzureRepo struct {
	Name string `json:"name"`
}

func RepoExists(orgURL, project, name string) (bool, error) {
	cmd := exec.Command("az", "repos", "show",
		"--repository", name,
		"--organization", orgURL,
		"--project", project,
		"--output", "json")

	out, err := cmd.Output()
	if err != nil {
		return false, nil // assume not exists
	}

	var repo AzureRepo
	if err := json.Unmarshal(out, &repo); err != nil {
		return false, nil
	}

	return repo.Name == name, nil
}

func CreateRepo(orgURL, project, name string) error {
	cmd := exec.Command("az", "repos", "create",
		"--name", name,
		"--project", project,
		"--organization", orgURL,
		"--output", "json")

	cmd.Stdout = nil
	cmd.Stderr = nil

	return cmd.Run()
}

func GetRepoRemoteURL(orgURL, project, repo string) (string, error) {

	cmd := exec.Command("az", "repos", "show",
		"--organization", orgURL,
		"--project", project,
		"--repository", repo,
		"--query", "remoteUrl",
		"-o", "tsv",
	)

	out, err := cmd.Output()
	if err != nil {
		return "", err
	}

	return strings.TrimSpace(string(out)), nil
}

func GetRepoNameByID_AzRepos(orgURL, project, repoID string) (string, error) {
	cmd := exec.Command("az", "repos", "show",
		"--organization", orgURL,
		"--project", project,
		"--id", repoID,
		"--query", "name",
		"-o", "tsv",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("az repos show --id failed: %w\n%s", err, string(out))
	}

	name := strings.TrimSpace(string(out))
	return name, nil
}

func GetRepoNameByID_Rest(orgURL, project, repoID, resourceGUID string) (string, error) {
	uri := fmt.Sprintf("%s/%s/_apis/git/repositories/%s?api-version=7.1", orgURL, project, repoID)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("az rest git repo by id failed: %w\n%s", err, string(out))
	}

	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		return "", fmt.Errorf("parse git repo JSON failed: %w\nRaw:\n%s", err, string(out))
	}

	if n, ok := obj["name"].(string); ok {
		return n, nil
	}
	return "", fmt.Errorf("git repo response has no name. Raw:\n%s", string(out))
}

func ResolveSourceRepoNameByID(orgURL, project, repoID, resourceGUID string) (string, error) {
	// try az repos show first
	if name, err := GetRepoNameByID_AzRepos(orgURL, project, repoID); err == nil && name != "" {
		return name, nil
	}
	// fallback to REST
	return GetRepoNameByID_Rest(orgURL, project, repoID, resourceGUID)
}

// Get repo name by repo ID (GUID) using az repos show --id
// Works across org/project as long as you call it against the correct org/project.
func GetRepoNameByID(orgURL, project, repoID string) (string, error) {
	cmd := exec.Command("az", "repos", "show",
		"--organization", orgURL,
		"--project", project,
		"--id", repoID,
		"--query", "name",
		"-o", "tsv",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("az repos show --id failed: %w\n%s", err, string(out))
	}
	return strings.TrimSpace(string(out)), nil
}

func GetRepoByID(orgURL, project, repoID, resourceGUID string) (*Repo, error) {
	uri := fmt.Sprintf("%s/%s/_apis/git/repositories/%s?api-version=7.1", orgURL, project, repoID)

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return nil, err
	}

	var r Repo
	if err := json.Unmarshal(out, &r); err != nil {
		return nil, fmt.Errorf("parse repo json failed: %w\nRaw:\n%s", err, string(out))
	}
	return &r, nil
}

func azRest(method, uri, resourceGUID string) ([]byte, error) {
	args := []string{"rest", "--method", method, "--uri", uri, "--output", "json", "--only-show-errors"}
	if strings.TrimSpace(resourceGUID) != "" {
		args = append(args, "--resource", resourceGUID)
	}
	cmd := exec.Command("az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest failed: %w\n%s", err, string(out))
	}
	return out, nil
}

func azRestWithBody(method, uri, resourceGUID string, body string) ([]byte, error) {
	args := []string{
		"rest",
		"--method", method,
		"--uri", uri,
		"--headers", "Content-Type=application/json",
		"--body", body,
		"--output", "json",
		"--only-show-errors",
	}
	if strings.TrimSpace(resourceGUID) != "" {
		args = append(args, "--resource", resourceGUID)
	}

	cmd := exec.Command("az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest failed: %w\n%s", err, string(out))
	}
	return out, nil
}

func ExtractOrgName(orgURL string) (string, error) {
	s := strings.TrimSpace(strings.TrimRight(orgURL, "/"))
	s = strings.TrimPrefix(s, "https://dev.azure.com/")
	s = strings.TrimPrefix(s, "http://dev.azure.com/")
	s = strings.TrimPrefix(s, "https://vssps.dev.azure.com/")
	s = strings.TrimPrefix(s, "http://vssps.dev.azure.com/")

	if i := strings.Index(s, "/"); i >= 0 {
		s = s[:i]
	}
	if s == "" {
		return "", fmt.Errorf("could not extract org name from url: %s", orgURL)
	}
	return s, nil
}
