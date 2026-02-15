package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupRelSourceOrg string
var backupRelSourceProject string
var backupRelDefinitions []string
var backupRelAdoResourceGUID string

var backupReleaseDefinitionsCmd = &cobra.Command{
	Use:   "backup-release-definitions",
	Short: "Backup classic release definitions locally (JSON) from Azure DevOps",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		orgName, orgCfg, err := cfg.ResolveOrganizationWithName(backupRelSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, orgName, orgCfg, backupRelSourceProject, "release-definitions")

		return internal.BackupReleaseDefinitions(
			orgCfg.URL,
			backupRelSourceProject,
			bkp,
			backupRelDefinitions,
			backupRelAdoResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupReleaseDefinitionsCmd)

	backupReleaseDefinitionsCmd.Flags().StringVar(&backupRelSourceOrg, "source-org", "", "Source organization")
	backupReleaseDefinitionsCmd.Flags().StringVar(&backupRelSourceProject, "source-project", "", "Source project")
	backupReleaseDefinitionsCmd.Flags().StringSliceVar(&backupRelDefinitions, "definitions", []string{}, "Release definition names or 'all'")
	backupReleaseDefinitionsCmd.Flags().StringVar(&backupRelAdoResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	backupReleaseDefinitionsCmd.MarkFlagRequired("source-org")
	backupReleaseDefinitionsCmd.MarkFlagRequired("source-project")
	backupReleaseDefinitionsCmd.MarkFlagRequired("definitions")
	backupReleaseDefinitionsCmd.MarkFlagRequired("ado-resource-guid")
}
