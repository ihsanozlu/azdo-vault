package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupYamlSourceOrg string
var backupYamlSourceProject string
var backupYamlPipelines []string
var backupYamlAdoResourceGUID string

var backupYamlPipelinesCmd = &cobra.Command{
	Use:   "backup-yaml-pipelines",
	Short: "Backup YAML pipelines locally (JSON) from Azure DevOps",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(backupYamlSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, backupYamlSourceProject, "yaml-pipelines")

		return internal.BackupYamlPipelines(
			sourceOrgCfg.URL,
			backupYamlSourceProject,
			bkp,
			backupYamlPipelines,
			backupYamlAdoResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupYamlPipelinesCmd)

	backupYamlPipelinesCmd.Flags().StringVar(
		&backupYamlSourceOrg,
		"source-org",
		"",
		"Source organization",
	)
	backupYamlPipelinesCmd.Flags().StringVar(
		&backupYamlSourceProject,
		"source-project",
		"",
		"Source project",
	)
	backupYamlPipelinesCmd.Flags().StringSliceVar(
		&backupYamlPipelines,
		"pipelines",
		[]string{},
		"Pipeline names or 'all'",
	)
	backupYamlPipelinesCmd.Flags().StringVar(
		&backupYamlAdoResourceGUID,
		"ado-resource-guid",
		"",
		"Azure DevOps AAD resource GUID (required for az rest)",
	)

	backupYamlPipelinesCmd.MarkFlagRequired("source-org")
	backupYamlPipelinesCmd.MarkFlagRequired("source-project")
	backupYamlPipelinesCmd.MarkFlagRequired("pipelines")
	backupYamlPipelinesCmd.MarkFlagRequired("ado-resource-guid")
}
