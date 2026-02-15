package internal

import (
	"encoding/json"
	"fmt"
	"strings"
)

type WikiListResponse struct {
	Count int    `json:"count"`
	Value []Wiki `json:"value"`
}

type Wiki struct {
	ID           string         `json:"id,omitempty"`
	Name         string         `json:"name,omitempty"`
	Type         string         `json:"type,omitempty"` // "projectWiki" or "codeWiki"
	RepositoryID string         `json:"repositoryId,omitempty"`
	MappedPath   string         `json:"mappedPath,omitempty"`
	ProjectID    string         `json:"projectId,omitempty"`
	Raw          map[string]any `json:"-"`
}

func (w *Wiki) UnmarshalJSON(data []byte) error {
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	w.Raw = m

	if s, ok := m["id"].(string); ok {
		w.ID = s
	}
	if s, ok := m["name"].(string); ok {
		w.Name = s
	}
	if s, ok := m["type"].(string); ok {
		w.Type = s
	}
	if s, ok := m["repositoryId"].(string); ok {
		w.RepositoryID = s
	}
	if s, ok := m["mappedPath"].(string); ok {
		w.MappedPath = s
	}
	if s, ok := m["projectId"].(string); ok {
		w.ProjectID = s
	}
	return nil
}

func (w Wiki) MarshalJSON() ([]byte, error) {
	if w.Raw != nil {
		return json.Marshal(w.Raw)
	}
	type alias Wiki
	return json.Marshal(alias(w))
}

func ListWikis(orgURL, project, resourceGUID string) ([]Wiki, error) {
	uri := fmt.Sprintf("%s/%s/_apis/wiki/wikis?api-version=7.1", strings.TrimRight(orgURL, "/"), project)

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return nil, fmt.Errorf("list wikis failed: %w", err)
	}

	var resp WikiListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing wikis json: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

// CreateWiki creates a ProjectWiki or CodeWiki.
// For CodeWiki you MUST send repositoryId + mappedPath (and optionally version).
func CreateWiki(orgURL, project, resourceGUID string, payload map[string]any) (*Wiki, error) {
	uri := fmt.Sprintf("%s/%s/_apis/wiki/wikis?api-version=7.1", strings.TrimRight(orgURL, "/"), project)

	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	out, err := azRestWithBody("post", uri, resourceGUID, string(bodyBytes))
	if err != nil {
		return nil, fmt.Errorf("create wiki failed: %w", err)
	}

	var created Wiki
	if err := json.Unmarshal(out, &created); err != nil {
		return nil, fmt.Errorf("failed parsing created wiki json: %w\nRaw:\n%s", err, string(out))
	}
	return &created, nil
}

// ---- small helpers ----

func IsProjectWiki(w Wiki) bool {
	return strings.EqualFold(w.Type, "projectWiki")
}

func IsCodeWiki(w Wiki) bool {
	return strings.EqualFold(w.Type, "codeWiki")
}

func FindWikiByName(wikis []Wiki, name string) *Wiki {
	for _, w := range wikis {
		if strings.EqualFold(w.Name, name) {
			return &w
		}
	}
	return nil
}
