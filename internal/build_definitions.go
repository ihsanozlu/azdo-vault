package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type BuildDefinitionListResponse struct {
	Count int               `json:"count"`
	Value []BuildDefinition `json:"value"`
}

type BuildDefinition struct {
	Id       int            `json:"id,omitempty"`
	Name     string         `json:"name"`
	Path     string         `json:"path,omitempty"`
	Revision int            `json:"revision,omitempty"`
	Raw      map[string]any `json:"-"`
}

type TaskAgentQueue struct {
	ID   int    `json:"id"`
	Name string `json:"name"`
}

type TaskAgentQueueListResponse struct {
	Count int              `json:"count"`
	Value []TaskAgentQueue `json:"value"`
}

func (b *BuildDefinition) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	b.Raw = m
	if v, ok := m["id"].(float64); ok {
		b.Id = int(v)
	}
	if v, ok := m["name"].(string); ok {
		b.Name = v
	}
	if v, ok := m["path"].(string); ok {
		b.Path = v
	}
	if v, ok := m["revision"].(float64); ok {
		b.Revision = int(v)
	}
	return nil
}

func (b BuildDefinition) MarshalJSON() ([]byte, error) {
	if b.Raw != nil {
		return json.Marshal(b.Raw)
	}
	type alias BuildDefinition
	return json.Marshal(alias(b))
}

// -----------------------------
// REST helpers
// -----------------------------

func ListBuildDefinitions(orgURL, project, resourceGUID string) ([]BuildDefinition, error) {
	uri := fmt.Sprintf("%s/%s/_apis/build/definitions?api-version=7.1", orgURL, project)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest list build definitions failed: %w\n%s", err, string(out))
	}

	var resp BuildDefinitionListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func GetBuildDefinition(orgURL, project, resourceGUID string, id int) (*BuildDefinition, error) {
	uri := fmt.Sprintf("%s/%s/_apis/build/definitions/%d?api-version=7.1", orgURL, project, id)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest get build definition failed: %w\n%s", err, string(out))
	}

	var def BuildDefinition
	if err := json.Unmarshal(out, &def); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}
	return &def, nil
}

func sanitizeBuildDefinitionForCreate(raw map[string]any) {
	// server-managed
	delete(raw, "id")
	delete(raw, "revision")
	delete(raw, "createdDate")
	delete(raw, "createdBy")
	delete(raw, "authoredBy")
	delete(raw, "queueStatus")

	// some exports include project objects / links
	delete(raw, "_links")
	delete(raw, "uri")
	delete(raw, "url")
	delete(raw, "project")
}

// NOTE: def.Raw must be ready (repo remapped etc.) before calling this.
func CreateBuildDefinition(orgURL, project, resourceGUID string, def BuildDefinition) (int, error) {
	if def.Raw == nil {
		b, _ := json.Marshal(def)
		_ = json.Unmarshal(b, &def.Raw)
	}

	sanitizeBuildDefinitionForCreate(def.Raw)

	bodyBytes, err := json.Marshal(def.Raw)
	if err != nil {
		return 0, err
	}

	uri := fmt.Sprintf("%s/%s/_apis/build/definitions?api-version=7.1", orgURL, project)

	// DEBUG: dump POST body
	// if os.Getenv("ADO_DUMP_BUILDDEF_BODY") == "1" {
	// 	_ = os.WriteFile(
	// 		fmt.Sprintf("/tmp/builddef_post_%s.json", strings.ReplaceAll(def.Name, " ", "_")),
	// 		bodyBytes,
	// 		0644,
	// 	)
	// 	fmt.Println("📝 dumped build definition POST body to /tmp for:", def.Name)
	// 	fmt.Println("DEBUG: ADO_DUMP_BUILDDEF_BODY is enabled, writing file for:", def.Name)
	// }

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
		return 0, fmt.Errorf("az rest create build definition failed: %w\n%s", err, string(out))
	}

	var created map[string]any
	if err := json.Unmarshal(out, &created); err != nil {
		return 0, fmt.Errorf("failed parsing created JSON: %w\nRaw:\n%s", err, string(out))
	}
	if idf, ok := created["id"].(float64); ok {
		return int(idf), nil
	}
	return 0, fmt.Errorf("created build definition but no id returned. Raw:\n%s", string(out))
}

func FindBuildDefinitionByName(orgURL, project, resourceGUID, name string) (*BuildDefinition, error) {
	list, err := ListBuildDefinitions(orgURL, project, resourceGUID)
	if err != nil {
		return nil, err
	}
	for _, d := range list {
		if d.Name == name {
			return &d, nil
		}
	}
	return nil, nil
}

// -----------------------------
// Repo resolution / mapping
// -----------------------------

func extractBuildRepoNameAndID(def map[string]any) (repoName, repoID string) {
	repo, ok := def["repository"].(map[string]any)
	if !ok {
		return "", ""
	}

	if s, ok := repo["name"].(string); ok && s != "" {
		repoName = s
	}
	if s, ok := repo["id"].(string); ok && s != "" {
		repoID = s
	}

	// Sometimes it’s in properties
	if repoName == "" {
		if props, ok := repo["properties"].(map[string]any); ok {
			if s, ok := props["repositoryName"].(string); ok && s != "" {
				repoName = s
			}
			if repoName == "" {
				if s, ok := props["name"].(string); ok && s != "" {
					repoName = s
				}
			}
		}
	}

	return repoName, repoID
}

func remapBuildDefinitionRepo(
	def map[string]any,
	sourceOrgURL, sourceProject string,
	targetRepoIDByName map[string]string,
) (string, string, error) {

	repoName, repoID := extractBuildRepoNameAndID(def)

	// If the exported def doesn't contain repo name, resolve from source repo id
	if repoName == "" && repoID != "" {
		n, err := GetRepoNameByID(sourceOrgURL, sourceProject, repoID)
		if err != nil {
			return "", "", fmt.Errorf("repo name not found (id='%s'): %w", repoID, err)
		}
		repoName = n
	}

	if repoName == "" {
		return "", "", fmt.Errorf("repo name is empty (repoID='%s')", repoID)
	}

	//targetID := targetRepoIDByName[repoName]
	targetID := targetRepoIDByName[strings.ToLower(repoName)]
	if targetID == "" {
		return repoName, "", fmt.Errorf("repo '%s' not found in target project", repoName)
	}

	// write back
	repoObj, ok := def["repository"].(map[string]any)
	if !ok || repoObj == nil {
		repoObj = map[string]any{}
		def["repository"] = repoObj
	}
	repoObj["id"] = targetID
	repoObj["name"] = repoName

	return repoName, targetID, nil
}

func ListTaskAgentQueues(orgURL, project, resourceGUID string) ([]TaskAgentQueue, error) {
	uri := fmt.Sprintf("%s/%s/_apis/distributedtask/queues?api-version=7.1", orgURL, project)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest list queues failed: %w\n%s", err, string(out))
	}

	var resp TaskAgentQueueListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing queues JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func parseKeyValuePairs(pairs []string) (map[string]string, error) {
	m := map[string]string{}
	for _, p := range pairs {
		if strings.TrimSpace(p) == "" {
			continue
		}
		parts := strings.SplitN(p, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid mapping '%s' (expected Source=Target)", p)
		}
		src := strings.TrimSpace(parts[0])
		tgt := strings.TrimSpace(parts[1])
		if src == "" || tgt == "" {
			return nil, fmt.Errorf("invalid mapping '%s' (empty side)", p)
		}
		m[src] = tgt
	}
	return m, nil
}

// tries to read queue.name from a build definition JSON
func extractBuildQueueName(def map[string]any) string {
	q, ok := def["queue"].(map[string]any)
	if !ok {
		return ""
	}
	if s, ok := q["name"].(string); ok && s != "" {
		return s
	}
	return ""
}

func setBuildQueueID(def map[string]any, id int) {
	q, ok := def["queue"].(map[string]any)
	if !ok || q == nil {
		q = map[string]any{}
		def["queue"] = q
	}
	q["id"] = id
}

// -----------------------------
// Backup / Restore
// -----------------------------

func BackupBuildDefinitions(orgURL, project, backupPath string, selected []string, resourceGUID string) error {
	list, err := ListBuildDefinitions(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}

	backupAll := len(selected) == 1 && selected[0] == "all"
	if len(list) == 0 {
		fmt.Println("No build definitions found")
		return nil
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, d := range list {
		if !backupAll && !contains(selected, d.Name) {
			continue
		}

		full, err := GetBuildDefinition(orgURL, project, resourceGUID, d.Id)
		if err != nil {
			return err
		}

		fp := filepath.Join(backupPath, full.Name+".json")
		data, err := json.MarshalIndent(full, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(fp, data, 0644); err != nil {
			return err
		}

		fmt.Println("✔ Backed up build definition:", full.Name)
	}
	return nil
}

func RestoreBuildDefinitionsFromBackup(
	sourceOrgURL, sourceProject string,
	targetOrgURL, targetProject, backupPath string,
	selected []string,
	resourceGUID string,
	queueMapPairs []string,
	defaultQueue string,
) error {

	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}
	restoreAll := len(selected) == 1 && selected[0] == "all"

	targetRepos, err := ListRepos(targetOrgURL, targetProject)
	if err != nil {
		return fmt.Errorf("failed to list target repos: %w", err)
	}
	targetRepoIDByName := map[string]string{}
	for _, r := range targetRepos {
		//targetRepoIDByName[r.Name] = r.Id
		targetRepoIDByName[strings.ToLower(r.Name)] = r.Id
	}

	targetQueues, err := ListTaskAgentQueues(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return err
	}
	targetQueueIDByName := map[string]int{}
	for _, q := range targetQueues {
		targetQueueIDByName[q.Name] = q.ID
	}

	queueMap, err := parseKeyValuePairs(queueMapPairs)
	if err != nil {
		return err
	}
	if defaultQueue != "" {
		if _, ok := targetQueueIDByName[defaultQueue]; !ok {
			return fmt.Errorf("default-queue '%s' not found in target project queues", defaultQueue)
		}
	}

	srcEPIDToName, tgtEPNameToID, err := BuildEndpointMaps(sourceOrgURL, sourceProject, targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return fmt.Errorf("failed to build service connection maps: %w", err)
	}
	srcVGIDToName, tgtVGNameToID, err := BuildVarGroupMaps(sourceOrgURL, sourceProject, targetOrgURL, targetProject)
	if err != nil {
		return fmt.Errorf("failed to build variable group maps: %w", err)
	}
	srcTGIDToName, tgtTGNameToID, err := BuildTaskGroupMaps(sourceOrgURL, sourceProject, targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return fmt.Errorf("failed to build task group maps: %w", err)
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

		var def BuildDefinition
		if err := json.Unmarshal(b, &def); err != nil {
			return err
		}

		existing, err := FindBuildDefinitionByName(targetOrgURL, targetProject, resourceGUID, def.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			fmt.Println("✔ Build definition exists, skipping:", def.Name)
			continue
		}

		if def.Raw == nil {
			tmp, _ := json.Marshal(def)
			_ = json.Unmarshal(tmp, &def.Raw)
		}

		repoName, _, err := remapBuildDefinitionRepo(def.Raw, sourceOrgURL, sourceProject, targetRepoIDByName)
		if err != nil {
			fmt.Printf("⚠ Skipping build definition '%s': repo remap failed: %s\n", def.Name, err.Error())
			continue
		}

		srcQueueName := extractBuildQueueName(def.Raw)
		targetQueueName := ""

		if srcQueueName != "" {
			if mapped, ok := queueMap[srcQueueName]; ok {
				targetQueueName = mapped
			} else {
				targetQueueName = srcQueueName
			}
		}

		targetQueueID := 0
		if targetQueueName != "" {
			if id, ok := targetQueueIDByName[targetQueueName]; ok {
				targetQueueID = id
			}
		}
		if targetQueueID == 0 && defaultQueue != "" {
			targetQueueID = targetQueueIDByName[defaultQueue]
			targetQueueName = defaultQueue
		}
		if targetQueueID == 0 {
			fmt.Printf("⚠ Skipping '%s': queue not resolved (source='%s'). Provide --queue-map or --default-queue.\n",
				def.Name, srcQueueName)
			continue
		}
		setBuildQueueID(def.Raw, targetQueueID)

		RemapBuildDefinitionRefsByName(
			def.Raw,
			srcEPIDToName,
			tgtEPNameToID,
			srcVGIDToName,
			tgtVGNameToID,
			srcTGIDToName,
			tgtTGNameToID,
		)

		fmt.Printf("Creating build definition: %s (repo='%s', queue: '%s' -> '%s')\n",
			def.Name, repoName, srcQueueName, targetQueueName)

		if _, err := CreateBuildDefinition(targetOrgURL, targetProject, resourceGUID, def); err != nil {
			fmt.Printf("⚠ Failed creating '%s'.\n%s\n", def.Name, err.Error())
			continue
		}
	}

	fmt.Println("✔ Build definitions restore finished (check warnings above)")
	return nil
}
