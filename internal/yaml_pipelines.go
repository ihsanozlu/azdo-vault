package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type PipelineListResponse struct {
	Count int        `json:"count"`
	Value []Pipeline `json:"value"`
}

type Pipeline struct {
	Id   int            `json:"id,omitempty"`
	Name string         `json:"name"`
	Raw  map[string]any `json:"-"`
}

func (p *Pipeline) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	p.Raw = m

	if v, ok := m["id"].(float64); ok {
		p.Id = int(v)
	}
	if v, ok := m["name"].(string); ok {
		p.Name = v
	}
	return nil
}

func (p Pipeline) MarshalJSON() ([]byte, error) {
	if p.Raw != nil {
		return json.Marshal(p.Raw)
	}
	type alias Pipeline
	return json.Marshal(alias(p))
}

// GET /_apis/pipelines?api-version=7.1
func ListPipelines(orgURL, project, resourceGUID string) ([]Pipeline, error) {
	uri := fmt.Sprintf("%s/%s/_apis/pipelines?api-version=7.1", orgURL, project)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest list pipelines failed: %w\n%s", err, string(out))
	}

	var resp PipelineListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

// GET /_apis/pipelines/{id}?api-version=7.1
func GetPipeline(orgURL, project, resourceGUID string, id int) (map[string]any, error) {
	uri := fmt.Sprintf("%s/%s/_apis/pipelines/%d?api-version=7.1", orgURL, project, id)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest get pipeline failed: %w\n%s", err, string(out))
	}

	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}
	return obj, nil
}

func isYamlPipeline(p map[string]any) bool {
	cfg, ok := p["configuration"].(map[string]any)
	if !ok {
		return false
	}
	if t, ok := cfg["type"].(string); ok && strings.EqualFold(t, "yaml") {
		return true
	}
	return false
}

func BackupYamlPipelines(orgURL, project, backupPath string, selected []string, resourceGUID string) error {
	list, err := ListPipelines(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}

	backupAll := len(selected) == 1 && selected[0] == "all"
	if len(list) == 0 {
		fmt.Println("No pipelines found")
		return nil
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, p := range list {
		if !backupAll && !contains(selected, p.Name) {
			continue
		}

		full, err := GetPipeline(orgURL, project, resourceGUID, p.Id)
		if err != nil {
			return err
		}

		if !isYamlPipeline(full) {
			continue
		}

		fp := filepath.Join(backupPath, p.Name+".json")
		data, err := json.MarshalIndent(full, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(fp, data, 0644); err != nil {
			return err
		}

		fmt.Println("✔ Backed up YAML pipeline:", p.Name)
	}

	return nil
}

func sanitizeYamlPipelineForCreate(obj map[string]any) map[string]any {
	name, _ := obj["name"].(string)

	folder := ""
	if f, ok := obj["folder"].(string); ok {
		folder = f
	}

	cfg, _ := obj["configuration"].(map[string]any)
	path, _ := cfg["path"].(string)

	repoName, _ := extractRepoNameAndID(obj)

	repo, _ := cfg["repository"].(map[string]any)
	repoType, _ := repo["type"].(string)
	if repoType == "" {
		repoType = "azureReposGit"
	}

	payload := map[string]any{
		"name": name,
	}
	if folder != "" {
		payload["folder"] = folder
	}
	payload["configuration"] = map[string]any{
		"type": "yaml",
		"path": path,
		"repository": map[string]any{
			"type": repoType,
			"name": repoName, // may still be empty; restored later by id→name resolve
		},
	}
	return payload
}

func CreateYamlPipeline(orgURL, project, resourceGUID string, payload map[string]any) (int, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	uri := fmt.Sprintf("%s/%s/_apis/pipelines?api-version=7.1", orgURL, project)

	cmd := exec.Command("az", "rest",
		"--method", "post",
		"--uri", uri,
		"--resource", resourceGUID,
		"--headers", "Content-Type=application/json",
		"--body", string(bodyBytes),
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("az rest create yaml pipeline failed: %w\n%s", err, string(out))
	}

	var created map[string]any
	if err := json.Unmarshal(out, &created); err != nil {
		return 0, fmt.Errorf("failed parsing created JSON: %w\nRaw:\n%s", err, string(out))
	}
	if idf, ok := created["id"].(float64); ok {
		return int(idf), nil
	}
	return 0, fmt.Errorf("created pipeline but no id returned. Raw:\n%s", string(out))
}

func RestoreYamlPipelinesFromBackup(
	sourceOrgURL, sourceProject string,
	targetOrgURL, targetProject, backupPath string,
	selected []string,
	resourceGUID string,
	targetRepos []Repo,
) error {
	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}

	restoreAll := len(selected) == 1 && selected[0] == "all"

	repoIdByName := map[string]string{}
	for _, r := range targetRepos {
		repoIdByName[r.Name] = r.Id
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		pipelineName := strings.TrimSuffix(f.Name(), ".json")
		if !restoreAll && !contains(selected, pipelineName) {
			continue
		}

		fp := filepath.Join(backupPath, f.Name())
		b, err := os.ReadFile(fp)
		if err != nil {
			return err
		}

		var full map[string]any
		if err := json.Unmarshal(b, &full); err != nil {
			return err
		}

		payload := sanitizeYamlPipelineForCreate(full)

		repoName, repoID := extractRepoNameAndID(full)

		if repoName == "" && repoID != "" {
			n, err := ResolveSourceRepoNameByID(sourceOrgURL, sourceProject, repoID, resourceGUID)
			if err != nil {
				fmt.Printf("⚠ Skipping pipeline '%s': could not resolve source repo name from id '%s'\n%s\n",
					pipelineName, repoID, err.Error())
				continue
			}
			repoName = n
		}

		if repoName == "" {
			fmt.Printf("⚠ Skipping pipeline '%s': repo name not found (id='%s')\n", pipelineName, repoID)
			continue
		}

		targetRepoID, ok := repoIdByName[repoName]
		if !ok {
			fmt.Printf("⚠ Skipping pipeline '%s': repo '%s' not found in target project\n", pipelineName, repoName)
			continue
		}

		cfg := payload["configuration"].(map[string]any)
		repo := cfg["repository"].(map[string]any)
		repo["id"] = targetRepoID
		repo["name"] = repoName

		existing, err := FindPipelineByName(targetOrgURL, targetProject, resourceGUID, pipelineName)
		if err != nil {
			return err
		}
		if existing != nil {
			fmt.Println("✔ YAML pipeline exists, skipping:", pipelineName)
			continue
		}

		fmt.Println("Creating YAML pipeline:", pipelineName)
		if _, err := CreateYamlPipeline(targetOrgURL, targetProject, resourceGUID, payload); err != nil {
			fmt.Printf("⚠ Failed creating YAML pipeline '%s'.\n%s\n", pipelineName, err.Error())
			continue
		}
	}

	fmt.Println("✔ YAML pipelines restore finished")
	return nil
}

func FindPipelineByName(orgURL, project, resourceGUID, name string) (*Pipeline, error) {
	list, err := ListPipelines(orgURL, project, resourceGUID)
	if err != nil {
		return nil, err
	}
	for _, p := range list {
		if p.Name == name {
			return &p, nil
		}
	}
	return nil, nil
}

func asMap(v any) (map[string]any, bool) {
	m, ok := v.(map[string]any)
	return m, ok
}

func asString(v any) (string, bool) {
	s, ok := v.(string)
	return s, ok
}

func extractRepoNameAndID(p map[string]any) (repoName string, repoID string) {
	cfg, _ := asMap(p["configuration"])
	repo, _ := asMap(cfg["repository"])

	if s, ok := asString(repo["id"]); ok && s != "" {
		repoID = s
	}

	// name
	if s, ok := asString(repo["name"]); ok && s != "" {
		return s, repoID
	}
	if s, ok := asString(repo["fullName"]); ok && s != "" {
		return s, repoID
	}

	if props, ok := asMap(repo["properties"]); ok {
		for _, k := range []string{"repositoryName", "fullName", "name"} {
			if s, ok := asString(props[k]); ok && s != "" {
				return s, repoID
			}
		}
	}

	return "", repoID
}
