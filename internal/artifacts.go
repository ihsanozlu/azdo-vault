package internal

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"
)

// -------------------- Models --------------------

type FeedsListResponse struct {
	Count int    `json:"count"`
	Value []Feed `json:"value"`
}

type Feed struct {
	ID                 string           `json:"id"`
	Name               string           `json:"name"`
	FullyQualifiedName string           `json:"fullyQualifiedName"`
	DefaultViewID      string           `json:"defaultViewId"`
	IsEnabled          bool             `json:"isEnabled"`
	HideDeleted        bool             `json:"hideDeletedPackageVersions"`
	UpstreamEnabled    bool             `json:"upstreamEnabled"`
	Project            *FeedProject     `json:"project,omitempty"`
	UpstreamSources    []UpstreamSource `json:"upstreamSources,omitempty"`
}

type FeedProject struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	Visibility string `json:"visibility"`
}

type UpstreamSource struct {
	ID                 string `json:"id"`
	Name               string `json:"name"`
	Protocol           string `json:"protocol"`
	Location           string `json:"location"`
	DisplayLocation    string `json:"displayLocation"`
	Status             string `json:"status"`
	UpstreamSourceType string `json:"upstreamSourceType"`
}

// Packages
type PackagesListResponse struct {
	Count int       `json:"count"`
	Value []Package `json:"value"`
}

type Package struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	ProtocolType string `json:"protocolType"`
	// Some responses include normalizedName, versions, etc. Add later if needed.
}

type PackageVersionsListResponse struct {
	Count int              `json:"count"`
	Value []PackageVersion `json:"value"`
}

type PackageVersion struct {
	ID       string `json:"id"`
	Version  string `json:"version"`
	IsListed bool   `json:"isListed"`
}

// -------------------- API --------------------
func ListFeeds(orgURL, project, resourceGUID string) ([]Feed, error) {
	orgName, err := ExtractOrgName(orgURL)
	if err != nil {
		return nil, err
	}

	// IMPORTANT: feeds.dev.azure.com (not dev.azure.com)
	uri := fmt.Sprintf("https://feeds.dev.azure.com/%s/%s/_apis/packaging/feeds?api-version=7.1-preview.1",
		orgName, url.PathEscape(project))

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return nil, err
	}

	var resp FeedsListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing feeds JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func ListPackages(orgURL, project, feedID, protocolType, resourceGUID string) ([]Package, error) {
	orgName, err := ExtractOrgName(orgURL)
	if err != nil {
		return nil, err
	}

	q := ""
	if strings.TrimSpace(protocolType) != "" {
		q = "&protocolType=" + url.QueryEscape(protocolType)
	}

	uri := fmt.Sprintf("https://feeds.dev.azure.com/%s/%s/_apis/packaging/feeds/%s/packages?api-version=7.1-preview.1%s",
		orgName, url.PathEscape(project), feedID, q)

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return nil, err
	}

	var resp PackagesListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing packages JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func ListPackageVersions(orgURL, project, feedID, packageID, resourceGUID string) ([]PackageVersion, error) {
	orgName, err := ExtractOrgName(orgURL)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("https://feeds.dev.azure.com/%s/%s/_apis/packaging/feeds/%s/packages/%s/versions?api-version=7.1-preview.1",
		orgName, url.PathEscape(project), feedID, packageID)

	out, err := azRest("get", uri, resourceGUID)
	if err != nil {
		return nil, err
	}

	var resp PackageVersionsListResponse
	if err := json.Unmarshal(out, &resp); err != nil {
		return nil, fmt.Errorf("failed parsing package versions JSON: %w\nRaw:\n%s", err, string(out))
	}
	return resp.Value, nil
}

func CreateFeed(orgURL, project, resourceGUID string, payload map[string]any) (*Feed, error) {
	orgName, err := ExtractOrgName(orgURL)
	if err != nil {
		return nil, err
	}

	uri := fmt.Sprintf("https://feeds.dev.azure.com/%s/%s/_apis/packaging/feeds?api-version=7.1-preview.1",
		orgName, url.PathEscape(project))

	bodyBytes, _ := json.Marshal(payload)
	out, err := azRestWithBody("post", uri, resourceGUID, string(bodyBytes))
	if err != nil {
		return nil, err
	}

	var created Feed
	if err := json.Unmarshal(out, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

func FindFeedByName(feeds []Feed, name string) *Feed {
	want := strings.ToLower(strings.TrimSpace(name))
	for i := range feeds {
		if strings.ToLower(strings.TrimSpace(feeds[i].Name)) == want {
			return &feeds[i]
		}
	}
	return nil
}
