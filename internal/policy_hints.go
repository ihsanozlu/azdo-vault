package internal

type PolicyBackupHints struct {
	RepoIDToName     map[string]string       `json:"repoIdToName,omitempty"`     // sourceRepoId -> repoName
	Identities       map[string]IdentityHint `json:"identities,omitempty"`       // sourceIdentityId -> hint (UPN)
	BuildDefinitions map[string]string       `json:"buildDefinitions,omitempty"` // sourceBuildId(str) -> name
}

type IdentityHint struct {
	UniqueName  string `json:"uniqueName,omitempty"` // usually email/UPN
	DisplayName string `json:"displayName,omitempty"`
}

type PolicyBackupFile struct {
	SourceOrganization string                `json:"sourceOrganization"`
	SourceProject      string                `json:"sourceProject"`
	Policies           []PolicyConfiguration `json:"policies"`
	Hints              PolicyBackupHints     `json:"hints"`
}

type PolicyConfiguration struct {
	ID         int            `json:"id,omitempty"`
	IsEnabled  bool           `json:"isEnabled"`
	IsBlocking bool           `json:"isBlocking"`
	Type       PolicyType     `json:"type"`
	Settings   map[string]any `json:"settings"`
}

type PolicyType struct {
	ID          string `json:"id"`
	DisplayName string `json:"displayName"`
}
