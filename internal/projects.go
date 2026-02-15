package internal

import (
	"encoding/json"
	"fmt"
	"os/exec"
)

type ProjectInfo struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

func GetProjectInfo(orgURL, projectName, resourceGUID string) (*ProjectInfo, error) {
	uri := fmt.Sprintf("%s/_apis/projects/%s?api-version=7.1", orgURL, projectName)

	cmd := exec.Command(
		"az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest get project failed: %w\n%s", err, string(out))
	}

	var p ProjectInfo
	if err := json.Unmarshal(out, &p); err != nil {
		return nil, fmt.Errorf("failed parsing project json: %w\nRaw:\n%s", err, string(out))
	}

	if p.Id == "" {
		return nil, fmt.Errorf("project id not found in response")
	}
	return &p, nil
}
