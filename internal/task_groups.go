package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type TaskGroupListResponse struct {
	Count int         `json:"count"`
	Value []TaskGroup `json:"value"`
}

type TaskGroup struct {
	Id           string           `json:"id,omitempty"`
	Name         string           `json:"name"`
	FriendlyName string           `json:"friendlyName,omitempty"`
	Description  string           `json:"description,omitempty"`
	Category     string           `json:"category,omitempty"`
	Version      map[string]any   `json:"version,omitempty"`
	Inputs       []map[string]any `json:"inputs,omitempty"`
	Tasks        []map[string]any `json:"tasks,omitempty"`
	Properties   map[string]any   `json:"properties,omitempty"`
	Disabled     bool             `json:"disabled,omitempty"`
	// Keep everything else without tightly coupling to schema:
	Raw map[string]any `json:"-"`
}

func (t *TaskGroup) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	t.Raw = m

	if v, ok := m["id"].(string); ok {
		t.Id = v
	}
	if v, ok := m["name"].(string); ok {
		t.Name = v
	}
	if v, ok := m["friendlyName"].(string); ok {
		t.FriendlyName = v
	}
	if v, ok := m["description"].(string); ok {
		t.Description = v
	}
	if v, ok := m["category"].(string); ok {
		t.Category = v
	}
	if v, ok := m["disabled"].(bool); ok {
		t.Disabled = v
	}

	// Optional complex fields (keep best-effort)
	if v, ok := m["version"].(map[string]any); ok {
		t.Version = v
	}
	if v, ok := m["properties"].(map[string]any); ok {
		t.Properties = v
	}
	if v, ok := m["inputs"].([]any); ok {
		var inputs []map[string]any
		for _, it := range v {
			if mm, ok := it.(map[string]any); ok {
				inputs = append(inputs, mm)
			}
		}
		t.Inputs = inputs
	}
	if v, ok := m["tasks"].([]any); ok {
		var tasks []map[string]any
		for _, it := range v {
			if mm, ok := it.(map[string]any); ok {
				tasks = append(tasks, mm)
			}
		}
		t.Tasks = tasks
	}

	return nil
}

func (t TaskGroup) MarshalJSON() ([]byte, error) {
	if t.Raw != nil {
		return json.Marshal(t.Raw)
	}

	type alias TaskGroup
	return json.Marshal(alias(t))
}

func ListTaskGroups(orgURL, project, resourceGUID string) ([]TaskGroup, error) {
	uri := fmt.Sprintf("%s/%s/_apis/distributedtask/taskgroups?api-version=7.1", orgURL, project)
	cmd := exec.Command("az", "rest", "--method", "get", "--uri", uri, "--resource", resourceGUID, "--output", "json", "--only-show-errors")

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest list taskgroups failed: %w\n%s", err, string(out))
	}

	var resp TaskGroupListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}
	return resp.Value, nil
}

func CreateTaskGroup(orgURL, project, resourceGUID string, tg TaskGroup) (string, error) {
	if tg.Raw == nil {
		// If not using Raw, create a Raw from marshaling
		b, _ := json.Marshal(tg)
		_ = json.Unmarshal(b, &tg.Raw)
	}

	// server-managed
	delete(tg.Raw, "id")
	delete(tg.Raw, "revision")
	delete(tg.Raw, "createdBy")
	delete(tg.Raw, "modifiedBy")
	delete(tg.Raw, "modifiedOn")
	delete(tg.Raw, "createdOn")

	// links / urls that can vary
	delete(tg.Raw, "_links")
	delete(tg.Raw, "url")
	delete(tg.Raw, "uri")

	bodyBytes, err := json.Marshal(tg.Raw)
	if err != nil {
		return "", err
	}

	uri := fmt.Sprintf("%s/%s/_apis/distributedtask/taskgroups?api-version=7.1", orgURL, project)
	cmd := exec.Command(
		"az", "rest",
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
		return "", fmt.Errorf("az rest create taskgroup failed: %w\n%s", err, string(out))
	}

	var created map[string]any
	if err := json.Unmarshal(out, &created); err != nil {
		return "", err
	}
	if id, ok := created["id"].(string); ok {
		return id, nil
	}
	return "", fmt.Errorf("created taskgroup but no id returned")
}

func FindTaskGroupByName(orgURL, project, resourceGUID, name string) (*TaskGroup, error) {
	list, err := ListTaskGroups(orgURL, project, resourceGUID)
	if err != nil {
		return nil, err
	}
	for _, g := range list {
		if g.Name == name {
			return &g, nil
		}
	}
	return nil, nil
}

func BackupTaskGroups(orgURL, project, backupPath string, selected []string, resourceGUID string) error {
	all, err := ListTaskGroups(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}

	backupAll := len(selected) == 1 && selected[0] == "all"
	if len(all) == 0 {
		fmt.Println("No task groups found")
		return nil
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, tg := range all {
		if !backupAll && !contains(selected, tg.Name) {
			continue
		}

		fp := filepath.Join(backupPath, tg.Name+".json")
		data, err := json.MarshalIndent(tg, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(fp, data, 0644); err != nil {
			return err
		}
		fmt.Println("✔ Backed up task group:", tg.Name)
	}
	return nil
}

// ------------------------------------------------------------
// Mapping helpers (service connections inside task group steps)
// ------------------------------------------------------------
func buildServiceConnectionMaps(
	sourceOrgURL, sourceProject,
	targetOrgURL, targetProject,
	resourceGUID string,
) (map[string]string, map[string]string, error) {

	src, err := ListServiceConnections(sourceOrgURL, sourceProject, resourceGUID)
	if err != nil {
		return nil, nil, fmt.Errorf("list source service connections failed: %w", err)
	}
	tgt, err := ListServiceConnections(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return nil, nil, fmt.Errorf("list target service connections failed: %w", err)
	}

	srcIDToName := map[string]string{}
	for _, e := range src {
		if e.Id != "" && e.Name != "" {
			srcIDToName[strings.ToLower(e.Id)] = e.Name
		}
	}

	tgtNameToID := map[string]string{}
	for _, e := range tgt {
		if e.Id != "" && e.Name != "" {
			tgtNameToID[e.Name] = e.Id
		}
	}

	return srcIDToName, tgtNameToID, nil
}

func looksLikeGUID(s string) bool {
	s = strings.TrimSpace(s)
	if len(s) != 36 {
		return false
	}
	// very small check: 8-4-4-4-12 with dashes
	return s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// RemapTaskGroupServiceConnections rewrites service connection IDs inside task group "tasks[].inputs"
func RemapTaskGroupServiceConnections(
	tgRaw map[string]any,
	srcEndpointIDToName map[string]string,
	tgtEndpointNameToID map[string]string,
) {
	tasksAny, _ := tgRaw["tasks"].([]any)
	for _, t := range tasksAny {
		task, _ := t.(map[string]any)
		inputs, _ := task["inputs"].(map[string]any)
		if inputs == nil {
			continue
		}

		for k, v := range inputs {
			s, ok := v.(string)
			if !ok || strings.TrimSpace(s) == "" {
				continue
			}

			// Some tasks store endpoint IDs under known keys, some store under custom keys.
			// Remap if (key indicates endpoint) OR (value looks like GUID and is found in src map).
			shouldTry := IsEndpointKey(k) || looksLikeGUID(s)
			if !shouldTry {
				continue
			}

			if epName, ok := srcEndpointIDToName[strings.ToLower(s)]; ok {
				if newID, ok := tgtEndpointNameToID[epName]; ok {
					inputs[k] = newID
				}
			}
		}
	}
}

// ------------------------------------------------------------
// Restore with remap
// ------------------------------------------------------------

// NEW SIGNATURE: needs source org/project so we can translate source endpoint IDs -> endpoint name -> target endpoint ID
func RestoreTaskGroupsFromBackup(
	sourceOrgURL, sourceProject string,
	targetOrgURL, targetProject string,
	backupPath string,
	selected []string,
	resourceGUID string,
) error {

	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}

	restoreAll := len(selected) == 1 && selected[0] == "all"

	srcEPIDToName, tgtEPNameToID, err := buildServiceConnectionMaps(
		sourceOrgURL, sourceProject,
		targetOrgURL, targetProject,
		resourceGUID,
	)
	if err != nil {
		return err
	}

	for _, f := range files {
		if !strings.HasSuffix(f.Name(), ".json") {
			continue
		}

		name := strings.TrimSuffix(f.Name(), ".json")
		if !restoreAll && !contains(selected, name) {
			continue
		}

		fp := filepath.Join(backupPath, f.Name())
		b, err := os.ReadFile(fp)
		if err != nil {
			return err
		}

		var tg TaskGroup
		if err := json.Unmarshal(b, &tg); err != nil {
			return err
		}

		if tg.Raw == nil {
			tmp, _ := json.Marshal(tg)
			_ = json.Unmarshal(tmp, &tg.Raw)
		}

		existing, err := FindTaskGroupByName(targetOrgURL, targetProject, resourceGUID, tg.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			fmt.Println("✔ Task group exists, skipping:", tg.Name)
			continue
		}

		RemapTaskGroupServiceConnections(tg.Raw, srcEPIDToName, tgtEPNameToID)

		fmt.Println("Creating task group:", tg.Name)
		if _, err := CreateTaskGroup(targetOrgURL, targetProject, resourceGUID, tg); err != nil {
			return err
		}
	}

	fmt.Println("✔ Task groups restored")
	return nil
}
