package cmd

import (
	"fmt"
	"path/filepath"

	"azdo-vault/internal"
)

func mustLoadConfig() (*internal.Config, error) {
	cfg, err := internal.LoadConfig()
	if err != nil {
		return nil, fmt.Errorf("configuration not found (run: azdo-vault configure add ...): %w", err)
	}
	return cfg, nil
}

// resolveSourceTarget resolves:
// - source org alias -> (sourceOrgName, sourceOrgCfg)
// - target org alias -> targetOrgCfg (defaults to sourceOrg if empty)
// - target project (defaults to source project if empty)
func resolveSourceTarget(
	cfg *internal.Config,
	sourceOrgAlias, sourceProject string,
	targetOrgAlias, targetProject string,
) (
	sourceOrgName string,
	sourceOrgCfg *internal.OrganizationConfig,
	resolvedTargetOrgAlias string,
	targetOrgCfg *internal.OrganizationConfig,
	resolvedTargetProject string,
	err error,
) {
	if sourceOrgAlias == "" {
		return "", nil, "", nil, "", fmt.Errorf("source-org is required")
	}
	if sourceProject == "" {
		return "", nil, "", nil, "", fmt.Errorf("source-project is required")
	}

	// source org
	srcName, srcCfg, err := cfg.ResolveOrganizationWithName(sourceOrgAlias)
	if err != nil {
		return "", nil, "", nil, "", err
	}

	// target org default = source org
	resTargetOrg := targetOrgAlias
	if resTargetOrg == "" {
		resTargetOrg = sourceOrgAlias
	}
	_, tgtCfg, err := cfg.ResolveOrganizationWithName(resTargetOrg)
	if err != nil {
		return "", nil, "", nil, "", err
	}

	// target project default = source project
	resTargetProject := targetProject
	if resTargetProject == "" {
		resTargetProject = sourceProject
	}

	return srcName, srcCfg, resTargetOrg, tgtCfg, resTargetProject, nil
}

func backupPath(
	cfg *internal.Config,
	orgName string,
	orgCfg *internal.OrganizationConfig,
	project string,
	kind string,
) string {
	_ = cfg // not used today; kept for future extensibility
	return filepath.Join(
		orgCfg.BackupRoot,
		orgName,
		project,
		kind,
	)
}
