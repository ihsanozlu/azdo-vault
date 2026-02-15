package internal

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ServiceEndpointListResponse struct {
	Count int               `json:"count"`
	Value []ServiceEndpoint `json:"value"`
}

type ServiceEndpoint struct {
	Id          string         `json:"id,omitempty"`
	Name        string         `json:"name"`
	Type        string         `json:"type,omitempty"`
	Url         string         `json:"url,omitempty"`
	Description string         `json:"description,omitempty"`
	IsShared    bool           `json:"isShared,omitempty"`
	IsReady     bool           `json:"isReady,omitempty"`
	Raw         map[string]any `json:"-"`
}

func (s *ServiceEndpoint) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	s.Raw = m

	if v, ok := m["id"].(string); ok {
		s.Id = v
	}
	if v, ok := m["name"].(string); ok {
		s.Name = v
	}
	if v, ok := m["type"].(string); ok {
		s.Type = v
	}
	if v, ok := m["url"].(string); ok {
		s.Url = v
	}
	if v, ok := m["description"].(string); ok {
		s.Description = v
	}
	if v, ok := m["isShared"].(bool); ok {
		s.IsShared = v
	}
	if v, ok := m["isReady"].(bool); ok {
		s.IsReady = v
	}

	return nil
}

func (s ServiceEndpoint) MarshalJSON() ([]byte, error) {
	if s.Raw != nil {
		return json.Marshal(s.Raw)
	}
	type alias ServiceEndpoint
	return json.Marshal(alias(s))
}

func ListServiceConnections(orgURL, project, resourceGUID string) ([]ServiceEndpoint, error) {
	uri := fmt.Sprintf("%s/%s/_apis/serviceendpoint/endpoints?api-version=7.1", orgURL, project)

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
		return nil, fmt.Errorf("az rest list service connections failed: %w\n%s", err, string(out))
	}

	var resp ServiceEndpointListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}

	return resp.Value, nil
}

func GetServiceConnection(orgURL, project, resourceGUID, endpointId string) (*ServiceEndpoint, error) {
	uri := fmt.Sprintf("%s/%s/_apis/serviceendpoint/endpoints/%s?api-version=7.1", orgURL, project, endpointId)

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
		return nil, fmt.Errorf("az rest show service connection failed: %w\n%s", err, string(out))
	}

	var ep ServiceEndpoint
	if err := json.Unmarshal(out, &ep); err != nil {
		return nil, fmt.Errorf("failed parsing JSON: %w\nRaw:\n%s", err, string(out))
	}

	return &ep, nil
}

func FindServiceConnectionByName(orgURL, project, resourceGUID, name string) (*ServiceEndpoint, error) {
	list, err := ListServiceConnections(orgURL, project, resourceGUID)
	if err != nil {
		return nil, err
	}
	for _, e := range list {
		if e.Name == name {
			return &e, nil
		}
	}
	return nil, nil
}

func CreateServiceConnection(orgURL, project, resourceGUID string, ep ServiceEndpoint, targetProjectID string) (string, error) {
	if ep.Raw == nil {
		b, _ := json.Marshal(ep)
		_ = json.Unmarshal(b, &ep.Raw)
	}

	sanitizeServiceConnectionForCreate(ep.Raw)

	// inject target project reference (required)
	ep.Raw["serviceEndpointProjectReferences"] = []map[string]any{
		{
			"projectReference": map[string]any{
				"id":   targetProjectID,
				"name": project, // target project name
			},
			"name":        ep.Name,
			"description": ep.Description,
		},
	}

	bodyBytes, err := json.Marshal(ep.Raw)
	if err != nil {
		return "", err
	}

	uri := fmt.Sprintf("%s/%s/_apis/serviceendpoint/endpoints?api-version=7.1", orgURL, project)

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
		return "", fmt.Errorf("az rest create service connection failed: %w\n%s", err, string(out))
	}

	var created map[string]any
	if err := json.Unmarshal(out, &created); err != nil {
		return "", fmt.Errorf("failed parsing created JSON: %w\nRaw:\n%s", err, string(out))
	}
	if id, ok := created["id"].(string); ok {
		return id, nil
	}
	return "", fmt.Errorf("created service connection but no id returned. Raw:\n%s", string(out))
}

func BackupServiceConnections(orgURL, project, backupPath string, selected []string, resourceGUID string) error {
	all, err := ListServiceConnections(orgURL, project, resourceGUID)
	if err != nil {
		return err
	}

	backupAll := len(selected) == 1 && selected[0] == "all"
	if len(all) == 0 {
		fmt.Println("No service connections found")
		return nil
	}

	if err := os.MkdirAll(backupPath, 0755); err != nil {
		return err
	}

	for _, e := range all {
		if !backupAll && !contains(selected, e.Name) {
			continue
		}

		// Get full details (often richer than list output)
		full, err := GetServiceConnection(orgURL, project, resourceGUID, e.Id)
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

		fmt.Println("✔ Backed up service connection:", full.Name)
	}

	return nil
}

func RestoreServiceConnectionsFromBackup(
	targetOrgURL, targetProject, backupPath string,
	selected []string,
	resourceGUID string,
) error {
	files, err := os.ReadDir(backupPath)
	if err != nil {
		return err
	}

	pinfo, err := GetProjectInfo(targetOrgURL, targetProject, resourceGUID)
	if err != nil {
		return err
	}

	restoreAll := len(selected) == 1 && selected[0] == "all"

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

		var ep ServiceEndpoint
		if err := json.Unmarshal(b, &ep); err != nil {
			return err
		}

		existing, err := FindServiceConnectionByName(targetOrgURL, targetProject, resourceGUID, ep.Name)
		if err != nil {
			return err
		}
		if existing != nil {
			fmt.Println("✔ Service connection exists, skipping:", ep.Name)
			continue
		}

		fmt.Println("Creating service connection:", ep.Name)
		_, err = CreateServiceConnection(targetOrgURL, targetProject, resourceGUID, ep, pinfo.Id)
		if err != nil {
			// Most common reason: missing secrets / auth params
			fmt.Printf("⚠ Failed to create '%s'. Likely needs manual re-auth / secrets.\n%s\n", ep.Name, err.Error())
			continue
		}
	}

	fmt.Println("✔ Service connections restore finished (check warnings above)")
	return nil
}

func sanitizeServiceConnectionForCreate(epRaw map[string]any) {
	delete(epRaw, "id")
	delete(epRaw, "createdBy")
	delete(epRaw, "creationDate")
	delete(epRaw, "modifiedBy")
	delete(epRaw, "modifiedOn")
	delete(epRaw, "owner")

	// These cause scope errors across orgs
	delete(epRaw, "administratorsGroup")
	delete(epRaw, "readersGroup")

	// We'll REBUILD serviceEndpointProjectReferences for the target project.
	delete(epRaw, "serviceEndpointProjectReferences")
}
