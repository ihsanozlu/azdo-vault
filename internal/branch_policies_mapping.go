package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// SanitizePolicyForCreate removes server-managed fields and returns a deep-copied payload.
func SanitizePolicyForCreate(full map[string]any) map[string]any {
	payload := deepCopyMapPolicy(full)

	// server-managed / read-only common fields
	delete(payload, "id")
	delete(payload, "revision")
	delete(payload, "url")
	delete(payload, "_links")
	delete(payload, "createdBy")
	delete(payload, "createdDate")
	delete(payload, "modifiedBy")
	delete(payload, "modifiedDate")

	// Some exports include these
	delete(payload, "isEnterpriseManaged")

	return payload
}

func deepCopyMapPolicy(in map[string]any) map[string]any {
	b, _ := json.Marshal(in)
	var out map[string]any
	_ = json.Unmarshal(b, &out)
	return out
}

func PolicyHitsAnyRepo(raw map[string]any, repoIDs map[string]bool) bool {
	if len(repoIDs) == 0 {
		return true
	}
	settings, _ := raw["settings"].(map[string]any)
	if settings == nil {
		return false
	}
	scopeAny, _ := settings["scope"].([]any)
	for _, s := range scopeAny {
		scope, _ := s.(map[string]any)
		if scope == nil {
			continue
		}
		if scope["repositoryId"] == nil {
			return true
		}
		if rid, ok := scope["repositoryId"].(string); ok && rid != "" {
			if repoIDs[strings.ToLower(rid)] {
				return true
			}
		}
	}
	return false
}

func PolicyShortLabel(raw map[string]any) string {
	t := policyTypeDisplayName(raw)
	if t == "" {
		t = "policy"
	}
	id := intFromAny(raw["id"])
	if id > 0 {
		return fmt.Sprintf("%s (id=%d)", t, id)
	}
	return t
}

func PolicySignature(raw map[string]any) string {
	t := policyTypeDisplayName(raw)
	settings, _ := raw["settings"].(map[string]any)

	scopeStr := ""
	if settings != nil {
		if scopes, ok := settings["scope"].([]any); ok {
			parts := []string{}
			for _, s := range scopes {
				m, _ := s.(map[string]any)
				if m == nil {
					continue
				}
				ref, _ := m["refName"].(string)
				rid := ""
				if m["repositoryId"] == nil {
					rid = "null"
				} else if s, ok := m["repositoryId"].(string); ok && s != "" {
					rid = strings.ToLower(s)
				}
				match, _ := m["matchKind"].(string)
				parts = append(parts, fmt.Sprintf("%s|%s|%s", rid, ref, match))
			}
			scopeStr = strings.Join(parts, ",")
		}
	}

	// include a tiny bit of settings for uniqueness (avoid duplicates between similar policies)
	key := fmt.Sprintf("%s||%s", t, scopeStr)

	// For min reviewers policy, include minimumApproverCount if present
	if settings != nil {
		if v, ok := settings["minimumApproverCount"].(float64); ok {
			key += fmt.Sprintf("||min=%d", int(v))
		}
		if v, ok := settings["allowDownvotes"].(bool); ok {
			key += fmt.Sprintf("||downvotes=%t", v)
		}
		if v, ok := settings["creatorVoteCounts"].(bool); ok {
			key += fmt.Sprintf("||creator=%t", v)
		}
	}

	return key
}

// RemapPolicyScopeRepoIDs rewrites settings.scope[].repositoryId
// source repoId -> source repoName -> target repoId
func RemapPolicyScopeRepoIDs(
	payload map[string]any,
	sourceOrgURL, sourceProject string,
	sourceRepoNameByID map[string]string,
	targetRepoIDByName map[string]string,
	resourceGUID string,
) error {
	settings, _ := payload["settings"].(map[string]any)
	if settings == nil {
		return nil
	}
	scopeAny, _ := settings["scope"].([]any)
	if len(scopeAny) == 0 {
		return nil
	}

	for _, s := range scopeAny {
		scope, _ := s.(map[string]any)
		if scope == nil {
			continue
		}

		// project-wide policies: repositoryId null => keep null
		if scope["repositoryId"] == nil {
			continue
		}

		srcRepoID, ok := scope["repositoryId"].(string)
		if !ok || strings.TrimSpace(srcRepoID) == "" {
			continue
		}

		srcRepoID = strings.ToLower(strings.TrimSpace(srcRepoID))
		repoName := sourceRepoNameByID[srcRepoID]
		if repoName == "" {
			return fmt.Errorf("source repo id '%s' not found in source project '%s' (repo deleted or not in project)", srcRepoID, sourceProject)
		}

		targetRepoID := targetRepoIDByName[strings.ToLower(repoName)]
		if targetRepoID == "" {
			return fmt.Errorf("target repo '%s' not found", repoName)
		}

		scope["repositoryId"] = targetRepoID
	}

	return nil
}

func ExtractIdentityIDs(raw map[string]any) []string {
	settings, _ := raw["settings"].(map[string]any)
	if settings == nil {
		return nil
	}

	out := map[string]bool{}

	if arr, ok := settings["requiredReviewerIds"].([]any); ok {
		for _, v := range arr {
			if s, ok := v.(string); ok && strings.TrimSpace(s) != "" {
				out[s] = true
			}
		}
	}

	if arr, ok := settings["requiredReviewers"].([]any); ok {
		for _, v := range arr {
			m, _ := v.(map[string]any)
			if m == nil {
				continue
			}
			if s, ok := m["id"].(string); ok && strings.TrimSpace(s) != "" {
				out[s] = true
			}
		}
	}

	ids := make([]string, 0, len(out))
	for k := range out {
		ids = append(ids, k)
	}
	return ids
}

func ExtractBuildDefinitionId(raw map[string]any) (int, bool) {
	settings, _ := raw["settings"].(map[string]any)
	if settings == nil {
		return 0, false
	}
	// buildDefinitionId is usually number in JSON
	if v, ok := settings["buildDefinitionId"].(float64); ok {
		return int(v), true
	}
	if v, ok := settings["buildDefinitionId"].(int); ok {
		return v, true
	}
	return 0, false
}

func RemapPolicyIdentityIDs(payload map[string]any, targetOrgURL, resourceGUID string) error {

	settings, _ := payload["settings"].(map[string]any)
	if settings == nil {
		return nil
	}

	needs := false
	if arr, ok := settings["requiredReviewerIds"].([]any); ok && len(arr) > 0 {
		needs = true
	}
	if arr, ok := settings["requiredReviewers"].([]any); ok && len(arr) > 0 {
		needs = true
	}

	hints := readPolicyHints(payload)
	if len(hints.Identities) == 0 {
		return nil
	}

	if needs && len(hints.Identities) == 0 {
		return fmt.Errorf("policy has required reviewers but backup has no identity hints; skipping to avoid TF402457")
	}
	if len(hints.Identities) == 0 {
		return nil
	}

	idMap := map[string]string{}
	for srcId, hint := range hints.Identities {
		tid, err := FindTargetIdentityIdByHint(targetOrgURL, hint, resourceGUID)
		if err != nil || strings.TrimSpace(tid) == "" {
			continue
		}
		idMap[srcId] = tid
	}
	if len(idMap) == 0 {
		return fmt.Errorf("no identities could be mapped (check that users/groups exist in target org)")
	}

	if arr, ok := settings["requiredReviewerIds"].([]any); ok {
		newArr := make([]any, 0, len(arr))
		for _, v := range arr {
			src, _ := v.(string)
			src = strings.TrimSpace(src)
			if src == "" {
				continue
			}
			mapped := idMap[src]
			if mapped == "" {
				continue
			}
			newArr = append(newArr, mapped)
		}

		if len(arr) > 0 && len(newArr) == 0 {
			return fmt.Errorf("all requiredReviewerIds could not be mapped; skipping policy")
		}

		settings["requiredReviewerIds"] = newArr
	}

	if arr, ok := settings["requiredReviewers"].([]any); ok {
		newArr := make([]any, 0, len(arr))
		for _, v := range arr {
			m, _ := v.(map[string]any)
			if m == nil {
				continue
			}
			src, _ := m["id"].(string)
			src = strings.TrimSpace(src)
			if src == "" {
				continue
			}
			mapped := idMap[src]
			if mapped == "" {
				continue
			}
			m["id"] = mapped
			newArr = append(newArr, m)
		}

		if len(arr) > 0 && len(newArr) == 0 {
			delete(settings, "requiredReviewers")
		} else {
			settings["requiredReviewers"] = newArr
		}
	}

	return nil
}

func RemapBuildValidationDefinition(payload map[string]any, targetOrgURL, targetProject, resourceGUID string) error {
	hints := readPolicyHints(payload)
	if len(hints.BuildDefinitions) == 0 {
		return nil
	}

	settings, _ := payload["settings"].(map[string]any)
	if settings == nil {
		return nil
	}

	var srcIdStr string
	if v, ok := settings["buildDefinitionId"].(float64); ok {
		srcIdStr = fmt.Sprintf("%d", int(v))
	} else if v, ok := settings["buildDefinitionId"].(int); ok {
		srcIdStr = fmt.Sprintf("%d", v)
	}
	if srcIdStr == "" {
		return nil
	}

	srcName := hints.BuildDefinitions[srcIdStr]
	if srcName == "" {
		return fmt.Errorf("no build definition hint for id=%s", srcIdStr)
	}

	targetId, err := FindBuildDefinitionIdByName(targetOrgURL, targetProject, srcName, resourceGUID)
	if err != nil {
		return err
	}

	settings["buildDefinitionId"] = targetId
	return nil
}

func FindBuildDefinitionIdByName(orgURL, project, name, resourceGUID string) (int, error) {
	// GET /_apis/build/definitions?name=<name>&api-version=7.1
	uri := fmt.Sprintf("%s/%s/_apis/build/definitions?name=%s&api-version=7.1",
		strings.TrimRight(orgURL, "/"), project, url.QueryEscape(name))

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return 0, err
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		return 0, err
	}
	val, _ := resp["value"].([]any)
	if len(val) == 0 {
		return 0, fmt.Errorf("target build definition not found by name: %s", name)
	}
	first, _ := val[0].(map[string]any)
	if first == nil {
		return 0, fmt.Errorf("invalid build definitions list response")
	}
	if idf, ok := first["id"].(float64); ok {
		return int(idf), nil
	}
	return 0, fmt.Errorf("could not parse target build definition id for name: %s", name)
}

func readPolicyHints(payload map[string]any) PolicyBackupHints {
	var hints PolicyBackupHints
	raw, _ := payload["_backupHints"]
	if raw == nil {
		return hints
	}

	b, _ := json.Marshal(raw)
	_ = json.Unmarshal(b, &hints)
	return hints
}
