package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ReleaseDefinitionListResponse struct {
	Count int              `json:"count"`
	Value []ReleaseSummary `json:"value"`
}

type ReleaseSummary struct {
	Id   int            `json:"id,omitempty"`
	Name string         `json:"name"`
	Raw  map[string]any `json:"-"`
}

func (r *ReleaseSummary) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	r.Raw = m
	if v, ok := m["id"].(float64); ok {
		r.Id = int(v)
	}
	if v, ok := m["name"].(string); ok {
		r.Name = v
	}
	return nil
}

func ListReleaseDefinitions(orgURL, project, resourceGUID string) ([]ReleaseSummary, error) {
	vsrmOrg := toVSRMBase(orgURL) // https://vsrm.dev.azure.com/{org}
	uri := fmt.Sprintf("%s/%s/_apis/release/definitions?api-version=7.1", vsrmOrg, project)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest list release definitions failed: %w\n%s", err, string(out))
	}

	var resp ReleaseDefinitionListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func GetReleaseDefinition(orgURL, project, resourceGUID string, id int) (map[string]any, error) {
	vsrmOrg := toVSRMBase(orgURL)
	uri := fmt.Sprintf("%s/%s/_apis/release/definitions/%d?api-version=7.1", vsrmOrg, project, id)

	cmd := exec.Command("az", "rest",
		"--method", "get",
		"--uri", uri,
		"--resource", resourceGUID,
		"--output", "json",
		"--only-show-errors",
	)

	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest get release definition failed: %w\n%s", err, string(out))
	}

	var obj map[string]any
	if err := json.Unmarshal(out, &obj); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}
	return obj, nil
}

func FindReleaseDefinitionByName(orgURL, project, resourceGUID, name string) (map[string]any, error) {
	list, err := ListReleaseDefinitions(orgURL, project, resourceGUID)
	if err != nil {
		return nil, err
	}
	for _, d := range list {
		if d.Name == name {
			return d.Raw, nil
		}
	}
	return nil, nil
}

func CreateReleaseDefinition(orgURL, project, resourceGUID string, payload map[string]any) (int, error) {
	vsrmOrg := toVSRMBase(orgURL)
	uri := fmt.Sprintf("%s/%s/_apis/release/definitions?api-version=7.1", vsrmOrg, project)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

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
		return 0, fmt.Errorf("az rest create release definition failed: %w\n%s", err, string(out))
	}

	var created map[string]any
	if err := json.Unmarshal(out, &created); err != nil {
		return 0, fmt.Errorf("failed parsing created JSON: %w\nRaw:\n%s", err, string(out))
	}

	if idf, ok := created["id"].(float64); ok {
		return int(idf), nil
	}
	return 0, fmt.Errorf("created release definition but no id returned. Raw:\n%s", string(out))
}

func BackupReleaseDefinitions(orgURL, project, backupPath string, selected []string, resourceGUID string) error {
	list, err := ListReleaseDefinitions(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}

	backupAll := len(selected) == 1 && selected[0] == "all"
	if len(list) == 0 {
		fmt.Println("No release definitions found")
		return nil
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, d := range list {
		if !backupAll && !contains(selected, d.Name) {
			continue
		}

		full, err := GetReleaseDefinition(orgURL, project, resourceGUID, d.Id)
		if err != nil {
			return err
		}

		fp := filepath.Join(backupPath, d.Name+".json")
		data, err := json.MarshalIndent(full, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(fp, data, 0644); err != nil {
			return err
		}

		fmt.Println("✔ Backed up release definition:", d.Name)
	}

	return nil
}

func RestoreReleaseDefinitionsFromBackup(
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

	targetQueues, err := ListTaskAgentQueues(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return err
	}
	targetQueueIDByName := map[string]int{}
	for _, q := range targetQueues {
		targetQueueIDByName[q.Name] = q.ID
	}

	sourceQueueIDToName, err := buildQueueIDToName(sourceOrgURL, sourceProject, resourceGUID)
	if err != nil {
		return fmt.Errorf("failed to list source queues: %w", err)
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

		var full map[string]any
		if err := json.Unmarshal(b, &full); err != nil {
			return err
		}

		existing, err := FindReleaseDefinitionByName(targetOrgURL, targetProject, resourceGUID, name)
		if err != nil {
			return err
		}
		if existing != nil {
			fmt.Println("✔ Release definition exists, skipping:", name)
			continue
		}

		payload := sanitizeReleaseDefinitionForCreate(full)

		// ✅ Remap artifact project/repo/build ids FIRST (before post)
		if err := RemapReleaseArtifacts(payload, targetOrgURL, targetProject, resourceGUID); err != nil {
			fmt.Printf("⚠ Skipping release '%s': artifact remap failed: %s\n", name, err.Error())
			continue
		}

		targetQName, targetQID, qerr := remapReleaseQueues(
			payload,
			sourceQueueIDToName,
			targetQueueIDByName,
			queueMap,
			defaultQueue,
		)

		if qerr != nil {
			fmt.Printf("⚠ Skipping release '%s': queue not resolved (%s). Provide --default-queue or fix --queue-map.\n", name, qerr.Error())
			continue
		}
		fmt.Printf("Queue mapped for release '%s': %s (id=%d)\n", name, targetQName, targetQID)

		RemapReleaseDefinitionRefsByName(
			payload,
			srcEPIDToName,
			tgtEPNameToID,
			srcVGIDToName,
			tgtVGNameToID,
			srcTGIDToName,
			tgtTGNameToID,
		)

		fmt.Println("Creating release definition:", name)
		if _, err := CreateReleaseDefinition(targetOrgURL, targetProject, resourceGUID, payload); err != nil {
			fmt.Printf("⚠ Failed creating release definition '%s'.\n%s\n", name, err.Error())
			continue
		}

	}

	fmt.Println("✔ Release definitions restore finished")
	return nil
}

// ---- helpers ----

func sanitizeReleaseDefinitionForCreate(full map[string]any) map[string]any {
	payload := deepCopyMap(full)

	// server-managed / read-only fields commonly present
	delete(payload, "id")
	delete(payload, "url")
	delete(payload, "_links")
	delete(payload, "createdBy")
	delete(payload, "createdOn")
	delete(payload, "modifiedBy")
	delete(payload, "modifiedOn")
	delete(payload, "revision")

	// some tenants include these
	delete(payload, "createdById")
	delete(payload, "modifiedById")

	//delete(payload, "source") // sometimes appears
	//delete(payload, "artifacts") // may include expanded refs; if you later want to restore artifacts, we can map them too

	return payload
}

func deepCopyMap(in map[string]any) map[string]any {
	b, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

// orgURL: https://dev.azure.com/{org}  -> https://vsrm.dev.azure.com/{org}
func toVSRMBase(orgURL string) string {
	u := strings.TrimRight(orgURL, "/")
	// supports both dev.azure.com/{org} and https://{org}.visualstudio.com
	if strings.Contains(u, "dev.azure.com/") {
		parts := strings.Split(u, "dev.azure.com/")
		if len(parts) == 2 && parts[1] != "" {
			return "https://vsrm.dev.azure.com/" + parts[1]
		}
	}
	// fallback: if user already gave vsrm or something else
	return strings.Replace(u, "https://dev.azure.com", "https://vsrm.dev.azure.com", 1)
}

// Azure DevOps release definition POST wants variableGroups as objects: [{ "id": 15 }]
func setReleaseVariableGroups(payload map[string]any, ids []int) {
	// out := make([]any, 0, len(ids))
	// for _, id := range ids {
	// 	out = append(out, map[string]any{"id": id})
	// }
	payload["variableGroups"] = ids
}

func readVarGroupIDAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	case map[string]any:
		if idf, ok := t["id"].(float64); ok {
			return int(idf)
		}
		if idi, ok := t["id"].(int); ok {
			return idi
		}
	}
	return 0
}

// Remap variableGroups + service endpoints + taskgroups in a release definition payload.
func RemapReleaseDefinitionRefsByName(
	payload map[string]any,
	srcEndpointIDToName map[string]string,
	tgtEndpointNameToID map[string]string,
	srcVarGroupIDToName map[int]string,
	tgtVarGroupNameToID map[string]int,
	srcTaskGroupIDToName map[string]string,
	tgtTaskGroupNameToID map[string]string,
) {
	// ReleaseDefinition.variableGroups is int[] (or may come expanded). We remap by name.
	if vgs, ok := payload["variableGroups"].([]any); ok {
		outIDs := make([]int, 0, len(vgs))

		for _, x := range vgs {
			srcID := readVarGroupIDAny(x)
			if srcID == 0 {
				continue
			}

			name, ok := srcVarGroupIDToName[srcID]
			if !ok || strings.TrimSpace(name) == "" {
				continue
			}

			if newID, ok := tgtVarGroupNameToID[strings.ToLower(name)]; ok && newID != 0 {
				outIDs = append(outIDs, newID)
			} else {
				// IMPORTANT: do NOT keep source IDs in target payload
				// (those IDs don't exist in target and can break create)
				fmt.Printf("⚠ %s: pipeline variable group '%s' (src id=%d) not found in target; dropping\n",
					fmt.Sprint(payload["name"]), name, srcID)
			}
		}

		setReleaseVariableGroups(payload, outIDs)
	}
	// ReleaseDefinitionEnvironment.variableGroups is also int[] in the API
	if envs, ok := payload["environments"].([]any); ok {
		for _, e := range envs {
			env, _ := e.(map[string]any)
			if env == nil {
				continue
			}
			if evgs, ok := env["variableGroups"].([]any); ok {
				outIDs := make([]int, 0, len(evgs))
				for _, x := range evgs {
					srcID := readVarGroupIDAny(x)
					if srcID == 0 {
						continue
					}

					name, ok := srcVarGroupIDToName[srcID]
					if !ok || strings.TrimSpace(name) == "" {
						continue
					}

					if newID, ok := tgtVarGroupNameToID[strings.ToLower(name)]; ok && newID != 0 {
						outIDs = append(outIDs, newID)
					} else {
						fmt.Printf("⚠ %s: stage variable group '%s' (src id=%d) not found in target; dropping\n",
							fmt.Sprint(payload["name"]), name, srcID)
					}
				}
				env["variableGroups"] = outIDs
			}
		}
	}

	envs, _ := payload["environments"].([]any)
	for _, e := range envs {
		env, _ := e.(map[string]any)

		dps, _ := env["deployPhases"].([]any)
		for _, dpAny := range dps {
			dp, _ := dpAny.(map[string]any)

			wts, _ := dp["workflowTasks"].([]any)
			for _, wtAny := range wts {
				wt, _ := wtAny.(map[string]any)

				if tid, ok := wt["taskId"].(string); ok && tid != "" {
					if name, ok := srcTaskGroupIDToName[strings.ToLower(tid)]; ok {
						if newID, ok := tgtTaskGroupNameToID[name]; ok {
							wt["taskId"] = newID
						}
					}
				}

				inputs, _ := wt["inputs"].(map[string]any)
				for k, v := range inputs {
					s, ok := v.(string)
					if !ok || s == "" {
						continue
					}
					if !IsEndpointKey(k) {
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
	}
}

func setReleaseQueueID(payload map[string]any, queueID int) {
	envs, _ := payload["environments"].([]any)
	for _, e := range envs {
		env, _ := e.(map[string]any)
		dps, _ := env["deployPhases"].([]any)
		for _, dpAny := range dps {
			dp, _ := dpAny.(map[string]any)
			di, _ := dp["deploymentInput"].(map[string]any)
			if di == nil {
				di = map[string]any{}
				dp["deploymentInput"] = di
			}
			di["queueId"] = queueID
		}
	}
}

// ---- Release queue mapping helpers ----

func buildQueueIDToName(orgURL, project, resourceGUID string) (map[int]string, error) {
	qs, err := ListTaskAgentQueues(orgURL, project, resourceGUID)
	if err != nil {
		return nil, err
	}
	m := map[int]string{}
	for _, q := range qs {
		if q.ID != 0 && q.Name != "" {
			m[q.ID] = q.Name
		}
	}
	return m, nil
}

func extractReleaseQueueIDs(payload map[string]any) []int {
	ids := []int{}

	envs, _ := payload["environments"].([]any)
	for _, e := range envs {
		env, _ := e.(map[string]any)
		dps, _ := env["deployPhases"].([]any)

		for _, dpAny := range dps {
			dp, _ := dpAny.(map[string]any)
			di, _ := dp["deploymentInput"].(map[string]any)
			if di == nil {
				continue
			}

			switch t := di["queueId"].(type) {
			case float64:
				if int(t) != 0 {
					ids = append(ids, int(t))
				}
			case int:
				if t != 0 {
					ids = append(ids, t)
				}
			}
		}
	}

	return ids
}

func remapReleaseQueues(
	payload map[string]any,
	sourceQueueIDToName map[int]string,
	targetQueueIDByName map[string]int,
	queueMap map[string]string,
	defaultQueue string,
) (string, int, error) {

	if defaultQueue != "" {
		id := targetQueueIDByName[defaultQueue]
		if id == 0 {
			return "", 0, fmt.Errorf("default-queue '%s' not found in target project queues", defaultQueue)
		}
		setReleaseQueueID(payload, id)
		return defaultQueue, id, nil
	}

	srcQueueIDs := extractReleaseQueueIDs(payload)
	if len(srcQueueIDs) == 0 {
		return "", 0, fmt.Errorf("no queueId found in release definition payload")
	}

	srcQName := sourceQueueIDToName[srcQueueIDs[0]]
	if srcQName == "" {
		return "", 0, fmt.Errorf("source queueId %d not found in source queues list", srcQueueIDs[0])
	}

	targetQName := srcQName
	if mapped, ok := queueMap[srcQName]; ok && mapped != "" {
		targetQName = mapped
	}

	targetQID := targetQueueIDByName[targetQName]
	if targetQID == 0 {
		return "", 0, fmt.Errorf("target queue '%s' not found in target project queues", targetQName)
	}

	setReleaseQueueID(payload, targetQID)
	return targetQName, targetQID, nil
}

func ensureStageRetentionPolicy(payload map[string]any, daysToKeep, releasesToKeep int, retainBuild bool) {
	envs, _ := payload["environments"].([]any)
	for _, e := range envs {
		env, _ := e.(map[string]any)
		if env == nil {
			continue
		}

		rp, ok := env["retentionPolicy"].(map[string]any)
		if !ok || rp == nil {
			env["retentionPolicy"] = map[string]any{
				"daysToKeep":     daysToKeep,
				"releasesToKeep": releasesToKeep,
				"retainBuild":    retainBuild,
			}
			continue
		}

		// fill missing keys if partially present
		if _, ok := rp["daysToKeep"]; !ok {
			rp["daysToKeep"] = daysToKeep
		}
		if _, ok := rp["releasesToKeep"]; !ok {
			rp["releasesToKeep"] = releasesToKeep
		}
		if _, ok := rp["retainBuild"]; !ok {
			rp["retainBuild"] = retainBuild
		}
	}
}
