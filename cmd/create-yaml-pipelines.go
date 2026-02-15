package cmd

import (
	"fmt"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreYamlSourceOrg string
var restoreYamlSourceProject string
var restoreYamlTargetOrg string
var restoreYamlTargetProject string
var restoreYamlPipelines []string
var restoreYamlAdoResourceGUID string

var createYamlPipelinesCmd = &cobra.Command{
	Use:   "create-yaml-pipelines",
	Short: "Create YAML pipelines in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreYamlSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := restoreYamlTargetOrg
		if targetOrg == "" {
			targetOrg = restoreYamlSourceOrg
		}

		targetProject := restoreYamlTargetProject
		if targetProject == "" {
			targetProject = restoreYamlSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreYamlSourceProject, "yaml-pipelines")

		// Need target repos to map repoName -> repoId
		targetRepos, err := internal.ListRepos(targetOrgCfg.URL, targetProject)
		if err != nil {
			return fmt.Errorf("failed listing target repos: %w", err)
		}

		return internal.RestoreYamlPipelinesFromBackup(
			sourceOrgCfg.URL,
			restoreYamlSourceProject,
			targetOrgCfg.URL,
			targetProject,
			bkp,
			restoreYamlPipelines,
			restoreYamlAdoResourceGUID,
			targetRepos,
		)
	},
}

func init() {
	rootCmd.AddCommand(createYamlPipelinesCmd)

	createYamlPipelinesCmd.Flags().StringVar(
		&restoreYamlSourceOrg,
		"source-org",
		"",
		"Source organization (where backup exists)",
	)
	createYamlPipelinesCmd.Flags().StringVar(
		&restoreYamlSourceProject,
		"source-project",
		"",
		"Source project (where backup exists)",
	)
	createYamlPipelinesCmd.Flags().StringVar(
		&restoreYamlTargetOrg,
		"target-org",
		"",
		"Target organization (defaults to source-org if omitted)",
	)
	createYamlPipelinesCmd.Flags().StringVar(
		&restoreYamlTargetProject,
		"target-project",
		"",
		"Target project (defaults to source-project if omitted)",
	)
	createYamlPipelinesCmd.Flags().StringSliceVar(
		&restoreYamlPipelines,
		"pipelines",
		[]string{},
		"Pipeline names or 'all'",
	)
	createYamlPipelinesCmd.Flags().StringVar(
		&restoreYamlAdoResourceGUID,
		"ado-resource-guid",
		"",
		"Azure DevOps AAD resource GUID (required for az rest)",
	)

	createYamlPipelinesCmd.MarkFlagRequired("source-org")
	createYamlPipelinesCmd.MarkFlagRequired("source-project")
	createYamlPipelinesCmd.MarkFlagRequired("pipelines")
	createYamlPipelinesCmd.MarkFlagRequired("ado-resource-guid")
}
