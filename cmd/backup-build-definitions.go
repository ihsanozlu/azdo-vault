package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var bldSourceOrg string
var bldSourceProject string
var bldNames []string
var bldResourceGUID string

var backupBuildDefsCmd = &cobra.Command{
	Use:   "backup-build-definitions",
	Short: "Backup classic build definitions locally (JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(bldSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, bldSourceProject, "build-definitions")

		return internal.BackupBuildDefinitions(
			sourceOrgCfg.URL,
			bldSourceProject,
			bkp,
			bldNames,
			bldResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupBuildDefsCmd)

	backupBuildDefsCmd.Flags().StringVar(&bldSourceOrg, "source-org", "", "Source organization")
	backupBuildDefsCmd.Flags().StringVar(&bldSourceProject, "source-project", "", "Source project")
	backupBuildDefsCmd.Flags().StringSliceVar(&bldNames, "definitions", []string{}, "Build definition names or 'all'")
	backupBuildDefsCmd.Flags().StringVar(&bldResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	backupBuildDefsCmd.MarkFlagRequired("source-org")
	backupBuildDefsCmd.MarkFlagRequired("source-project")
	backupBuildDefsCmd.MarkFlagRequired("definitions")
	backupBuildDefsCmd.MarkFlagRequired("ado-resource-guid")
}
