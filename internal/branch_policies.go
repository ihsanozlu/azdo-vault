package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type PolicyConfigListResponse struct {
	Count int            `json:"count"`
	Value []PolicyConfig `json:"value"`
}

type PolicyConfig struct {
	Id         int            `json:"id,omitempty"`
	Type       map[string]any `json:"type,omitempty"`
	IsEnabled  bool           `json:"isEnabled,omitempty"`
	IsBlocking bool           `json:"isBlocking,omitempty"`
	Raw        map[string]any `json:"-"`
}

func (p *PolicyConfig) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	p.Raw = m

	if v, ok := m["id"].(float64); ok {
		p.Id = int(v)
	}
	if v, ok := m["type"].(map[string]any); ok {
		p.Type = v
	}
	if v, ok := m["isEnabled"].(bool); ok {
		p.IsEnabled = v
	}
	if v, ok := m["isBlocking"].(bool); ok {
		p.IsBlocking = v
	}
	return nil
}

func (p PolicyConfig) MarshalJSON() ([]byte, error) {
	if p.Raw != nil {
		return json.Marshal(p.Raw)
	}
	type alias PolicyConfig
	return json.Marshal(alias(p))
}

// ListPolicyConfigurations lists ALL policy configs in a project.
// REST: GET /_apis/policy/configurations?api-version=7.1
func ListPolicyConfigurations(orgURL, project, resourceGUID string) ([]PolicyConfig, error) {
	uri := fmt.Sprintf("%s/%s/_apis/policy/configurations?api-version=7.1", strings.TrimRight(orgURL, "/"), project)

	args := []string{
		"rest",
		"--method", "get",
		"--uri", uri,
		"--output", "json",
		"--only-show-errors",
	}

	if strings.TrimSpace(resourceGUID) != "" {
		args = append(args, "--resource", resourceGUID)
	}

	cmd := exec.Command("az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("az rest list policy configurations failed: %w\n%s", err, string(out))
	}

	var resp PolicyConfigListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing policy configurations JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func CreatePolicyConfiguration(orgURL, project, resourceGUID string, payload map[string]any) (int, error) {
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return 0, err
	}

	uri := fmt.Sprintf("%s/%s/_apis/policy/configurations?api-version=7.1", strings.TrimRight(orgURL, "/"), project)

	args := []string{
		"rest",
		"--method", "post",
		"--uri", uri,
		"--headers", "Content-Type=application/json",
		"--body", string(bodyBytes),
		"--output", "json",
		"--only-show-errors",
	}
	if strings.TrimSpace(resourceGUID) != "" {
		args = append(args, "--resource", resourceGUID)
	}

	cmd := exec.Command("az", args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return 0, fmt.Errorf("az rest create policy configuration failed: %w\n%s", err, string(out))
	}

	var created map[string]any
	if err := json.Unmarshal(out, &created); err != nil {
		return 0, fmt.Errorf("failed parsing created policy JSON: %w\nRaw:\n%s", err, string(out))
	}
	if idf, ok := created["id"].(float64); ok {
		return int(idf), nil
	}
	return 0, fmt.Errorf("created policy but no id returned. Raw:\n%s", string(out))
}

func FindPolicyConfigBySignature(
	targetConfigs []PolicyConfig,
	signature string,
) *PolicyConfig {
	for _, c := range targetConfigs {
		if PolicySignature(c.Raw) == signature {
			return &c
		}
	}
	return nil
}

// BackupBranchPolicies writes each policy config as one JSON file.
// It optionally filters by repo IDs in scope.
func BackupBranchPolicies(
	orgURL, project, backupPath string,
	selectedRepos []string, // repo names or ["all"]
	resourceGUID string,
) error {

	repoIDs, err := resolveRepoIDsForFilter(orgURL, project, selectedRepos)
	if err != nil {
		return err
	}

	all, err := ListPolicyConfigurations(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}
	if len(all) == 0 {
		fmt.Println("No policy configurations found")
		return nil
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, pc := range all {
		if pc.Raw == nil {
			continue
		}

		if len(repoIDs) > 0 {
			if !PolicyHitsAnyRepo(pc.Raw, repoIDs) {
				continue
			}
		}

		hints := PolicyBackupHints{
			Identities:       map[string]IdentityHint{},
			BuildDefinitions: map[string]string{},
		}

		for _, id := range ExtractIdentityIDs(pc.Raw) {
			ih, err := GetIdentityById(orgURL, id, resourceGUID)
			if err == nil && ih != nil {
				hints.Identities[id] = *ih
			} else {
				fmt.Printf("⚠ backup: could not resolve identity id=%s policy=%s err=%v\n", id, PolicyShortLabel(pc.Raw), err)
			}
		}
		pc.Raw["_backupHints"] = hints

		if bldId, ok := ExtractBuildDefinitionId(pc.Raw); ok {
			// GET /_apis/build/definitions/{id}
			name, err := GetBuildDefinitionName(orgURL, project, bldId, resourceGUID)
			if err == nil && name != "" {
				hints.BuildDefinitions[fmt.Sprintf("%d", bldId)] = name
			}
		}

		fp := filepath.Join(backupPath, policyFilename(pc.Raw))
		data, err := json.MarshalIndent(pc, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(fp, data, 0644); err != nil {
			return err
		}

		fmt.Println("✔ Backed up policy:", PolicyShortLabel(pc.Raw))
	}

	return nil
}

func RestoreBranchPoliciesFromBackup(
	sourceOrgURL, sourceProject string,
	targetOrgURL, targetProject, backupPath string,
	selected []string, // filenames or "all"
	resourceGUID string,
) error {

	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}
	restoreAll := len(selected) == 1 && strings.EqualFold(selected[0], "all")

	// Build target repo name -> id map
	targetRepos, err := ListRepos(targetOrgURL, targetProject)
	if err != nil {
		return fmt.Errorf("failed to list target repos: %w", err)
	}
	targetRepoIDByName := map[string]string{}
	for _, r := range targetRepos {
		//targetRepoIDByName[r.Name] = r.Id
		targetRepoIDByName[strings.ToLower(r.Name)] = r.Id
	}

	// Build SOURCE repo id -> name map (once)
	sourceRepos, err := ListRepos(sourceOrgURL, sourceProject)
	if err != nil {
		return fmt.Errorf("failed to list source repos: %w", err)
	}
	sourceRepoNameByID := map[string]string{}
	for _, r := range sourceRepos {
		sourceRepoNameByID[strings.ToLower(r.Id)] = r.Name
	}

	// Load target existing policies once for "exists" check
	targetExisting, err := ListPolicyConfigurations(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return fmt.Errorf("failed listing target policies: %w", err)
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

		var pc PolicyConfig
		if err := json.Unmarshal(b, &pc); err != nil {
			return err
		}
		if pc.Raw == nil {
			tmp, _ := json.Marshal(pc)
			_ = json.Unmarshal(tmp, &pc.Raw)
		}

		// Build "signature" for dedupe (type + scopes + key settings)
		sig := PolicySignature(pc.Raw)
		if existing := FindPolicyConfigBySignature(targetExisting, sig); existing != nil {
			fmt.Println("✔ Policy exists, skipping:", PolicyShortLabel(pc.Raw))
			continue
		}

		payload := SanitizePolicyForCreate(pc.Raw)

		// identity mapping
		if err := RemapPolicyIdentityIDs(payload, targetOrgURL, resourceGUID); err != nil {
			fmt.Printf("⚠ Identity mapping failed for '%s': %s\n", PolicyShortLabel(payload), err.Error())
			continue
		}

		//DEBUG
		if s, ok := payload["settings"].(map[string]any); ok {
			fmt.Printf("DEBUG mapped requiredReviewerIds=%v\n", s["requiredReviewerIds"])
		}

		// build validation mapping
		if err := RemapBuildValidationDefinition(payload, targetOrgURL, targetProject, resourceGUID); err != nil {
			fmt.Printf("⚠ Build validation mapping warning for '%s': %s\n", PolicyShortLabel(payload), err.Error())
			// do NOT skip; try creating anyway
		}

		// repoId mapping inside settings.scope[]
		if err := RemapPolicyScopeRepoIDs(
			payload,
			sourceOrgURL, sourceProject,
			sourceRepoNameByID,
			targetRepoIDByName,
			resourceGUID,
		); err != nil {
			fmt.Printf("⚠ Skipping policy '%s': repo mapping failed: %s\n", PolicyShortLabel(pc.Raw), err.Error())
			continue
		}

		delete(payload, "_backupHints")

		fmt.Println("Creating policy:", PolicyShortLabel(payload))
		if _, err := CreatePolicyConfiguration(targetOrgURL, targetProject, resourceGUID, payload); err != nil {
			fmt.Printf("⚠ Failed creating policy '%s'.\n%s\n", PolicyShortLabel(payload), err.Error())
			continue
		}
	}

	fmt.Println("✔ Branch policies restore finished")
	return nil
}

// ---------- helpers ----------

func policyFilename(raw map[string]any) string {
	id := intFromAny(raw["id"])
	t := policyTypeDisplayName(raw)
	if t == "" {
		t = "policy"
	}
	t = safeFilePart(t)

	if id > 0 {
		return fmt.Sprintf("%d_%s.json", id, t)
	}
	return fmt.Sprintf("new_%s.json", t)
}

func safeFilePart(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "/", "-")
	s = strings.ReplaceAll(s, "\\", "-")
	s = strings.ReplaceAll(s, ":", "-")
	s = strings.ReplaceAll(s, "*", "-")
	s = strings.ReplaceAll(s, "?", "-")
	s = strings.ReplaceAll(s, "\"", "-")
	s = strings.ReplaceAll(s, "<", "-")
	s = strings.ReplaceAll(s, ">", "-")
	s = strings.ReplaceAll(s, "|", "-")
	re := regexp.MustCompile(`\s+`)
	s = re.ReplaceAllString(s, "_")
	if s == "" {
		return "policy"
	}
	return s
}

func policyTypeDisplayName(raw map[string]any) string {
	t, _ := raw["type"].(map[string]any)
	if t == nil {
		return ""
	}
	if s, ok := t["displayName"].(string); ok && s != "" {
		return s
	}
	return ""
}

func intFromAny(v any) int {
	switch t := v.(type) {
	case float64:
		return int(t)
	case int:
		return t
	default:
		return 0
	}
}

// If user passed --repos all => no filter (empty map)
func resolveRepoIDsForFilter(orgURL, project string, selectedRepos []string) (map[string]bool, error) {
	if len(selectedRepos) == 0 {
		return map[string]bool{}, nil
	}
	if len(selectedRepos) == 1 && strings.EqualFold(selectedRepos[0], "all") {
		return map[string]bool{}, nil
	}

	all, err := ListRepos(orgURL, project)
	if err != nil {
		return nil, err
	}

	want := map[string]bool{}
	for _, r := range selectedRepos {
		want[strings.ToLower(r)] = true
	}

	out := map[string]bool{}
	for _, r := range all {
		if want[strings.ToLower(r.Name)] {
			out[strings.ToLower(r.Id)] = true
		}
	}

	if len(out) == 0 {
		return nil, fmt.Errorf("no repos matched --repos input")
	}
	return out, nil
}

func GetIdentityById(orgURL, identityId, resourceGUID string) (*IdentityHint, error) {
	org := strings.TrimRight(orgURL, "/")
	org = strings.TrimPrefix(org, "https://dev.azure.com/")
	org = strings.TrimPrefix(org, "http://dev.azure.com/")

	uri := fmt.Sprintf("https://vssps.dev.azure.com/%s/_apis/identities?identityIds=%s&api-version=7.1-preview.1", org, identityId)

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return nil, err
	}

	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, err
	}

	val, _ := resp["value"].([]any)
	if len(val) == 0 {
		return nil, fmt.Errorf("identity not found for id=%s", identityId)
	}

	obj, _ := val[0].(map[string]any)
	if obj == nil {
		return nil, fmt.Errorf("invalid identity payload")
	}

	h := &IdentityHint{}

	// displayName fallbacks
	if s, ok := obj["displayName"].(string); ok && strings.TrimSpace(s) != "" {
		h.DisplayName = s
	}
	if h.DisplayName == "" {
		if s, ok := obj["providerDisplayName"].(string); ok && strings.TrimSpace(s) != "" {
			h.DisplayName = s
		}
	}
	if h.DisplayName == "" {
		if s, ok := obj["customDisplayName"].(string); ok && strings.TrimSpace(s) != "" {
			h.DisplayName = s
		}
	}

	// uniqueName fallbacks
	pick := func(x any) string {
		s, _ := x.(string)
		return strings.TrimSpace(s)
	}

	if s := pick(obj["uniqueName"]); s != "" {
		h.UniqueName = s
	}
	if h.UniqueName == "" {
		if s := pick(obj["signInAddress"]); s != "" {
			h.UniqueName = s
		}
	}
	if h.UniqueName == "" {
		if s := pick(obj["mailAddress"]); s != "" {
			h.UniqueName = s
		}
	}

	// IMPORTANT: properties fallback (this is usually where Account/UPN lives)
	if h.UniqueName == "" {
		if props, ok := obj["properties"].(map[string]any); ok && props != nil {
			// many identity APIs encode values as { "$value": "..." }
			getProp := func(key string) string {
				v := props[key]
				if m, ok := v.(map[string]any); ok && m != nil {
					if s := pick(m["$value"]); s != "" {
						return s
					}
				}
				return pick(v)
			}

			// try common keys
			for _, k := range []string{"Account", "Mail", "SignInAddress", "Email"} {
				if s := getProp(k); s != "" {
					h.UniqueName = s
					break
				}
			}
		}
	}

	// If both are empty, we can't map later — treat as error for backup hints.
	if h.UniqueName == "" && h.DisplayName == "" {
		return nil, fmt.Errorf("identity %s has no uniqueName/displayName in response", identityId)
	}

	return h, nil
}

func FindTargetIdentityIdByHint(targetOrgURL string, hint IdentityHint, resourceGUID string) (string, error) {
	// Best effort search by uniqueName first, fallback to displayName.
	// Identity search:
	// GET https://vssps.dev.azure.com/{org}/_apis/identities?searchFilter=General&filterValue=<value>&api-version=7.1-preview.1

	org := strings.TrimRight(targetOrgURL, "/")
	org = strings.TrimPrefix(org, "https://dev.azure.com/")
	org = strings.TrimPrefix(org, "http://dev.azure.com/")

	query := hint.UniqueName
	if strings.TrimSpace(query) == "" {
		query = hint.DisplayName
	}
	if strings.TrimSpace(query) == "" {
		return "", fmt.Errorf("empty identity hint")
	}

	uri := fmt.Sprintf("https://vssps.dev.azure.com/%s/_apis/identities?searchFilter=General&filterValue=%s&api-version=7.1-preview.1",
		org, url.QueryEscape(query))

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return "", err
	}

	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", err
	}

	val, _ := resp["value"].([]any)
	if len(val) == 0 {
		return "", fmt.Errorf("no target identity match for '%s'", query)
	}

	// pick best match: exact uniqueName if possible
	bestId := ""
	for _, v := range val {
		m, _ := v.(map[string]any)
		if m == nil {
			continue
		}
		id, _ := m["id"].(string)
		uniq, _ := m["uniqueName"].(string)
		disp, _ := m["displayName"].(string)

		if hint.UniqueName != "" && strings.EqualFold(uniq, hint.UniqueName) {
			return id, nil
		}
		// fallback to exact displayName
		if bestId == "" && hint.DisplayName != "" && strings.EqualFold(disp, hint.DisplayName) {
			bestId = id
		}
	}

	if bestId != "" {
		return bestId, nil
	}

	// otherwise first result
	first, _ := val[0].(map[string]any)
	if first != nil {
		if id, ok := first["id"].(string); ok && id != "" {
			return id, nil
		}
	}

	return "", fmt.Errorf("could not extract target identity id from search results")
}

func GetBuildDefinitionName(orgURL, project string, id int, resourceGUID string) (string, error) {
	uri := fmt.Sprintf("%s/%s/_apis/build/definitions/%d?api-version=7.1", strings.TrimRight(orgURL, "/"), project, id)
	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return "", err
	}
	var m map[string]any
	if err := json.Unmarshal(out, &m); err != nil {
		return "", err
	}
	if s, ok := m["name"].(string); ok {
		return s, nil
	}
	return "", fmt.Errorf("build definition name missing for id=%d", id)
}

func BuildTargetReposByName(repos []Repo) map[string]string {
	m := map[string]string{}
	for _, r := range repos {
		m[strings.ToLower(r.Name)] = r.Id // or r.Id depending your struct
	}
	return m
}

func ResolveIdentityIDByUPN(targetOrgURL, upn string) (string, error) {
	// targetOrgURL is https://dev.azure.com/{orgName}
	// vssps base: https://vssps.dev.azure.com/{orgName}
	orgName, err := ExtractOrgName(targetOrgURL)
	if err != nil {
		return "", err
	}

	uri := fmt.Sprintf("https://vssps.dev.azure.com/%s/_apis/identities?searchFilter=General&filterValue=%s&queryMembership=None&api-version=7.1-preview.1",
		orgName,
		url.QueryEscape(upn),
	)

	out, err := azRest("get", uri, "")
	if err != nil {
		return "", err
	}

	var resp struct {
		Value []struct {
			ID         string `json:"id"`
			UniqueName string `json:"uniqueName"`
		} `json:"value"`
	}
	if err := json.Unmarshal(out, &resp); err != nil {
		return "", err
	}

	// pick exact match on uniqueName, else first
	low := strings.ToLower(upn)
	for _, v := range resp.Value {
		if strings.ToLower(v.UniqueName) == low && v.ID != "" {
			return v.ID, nil
		}
	}
	if len(resp.Value) > 0 && resp.Value[0].ID != "" {
		return resp.Value[0].ID, nil
	}

	return "", fmt.Errorf("identity not found for '%s'", upn)
}
