package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupSCSourceOrg string
var backupSCSourceProject string
var backupSCNames []string
var backupSCAdoResourceGUID string

var backupServiceConnectionsCmd = &cobra.Command{
	Use:   "backup-service-connections",
	Short: "Backup service connections locally (JSON) from Azure DevOps",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(backupSCSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, backupSCSourceProject, "service-connections")

		return internal.BackupServiceConnections(
			sourceOrgCfg.URL,
			backupSCSourceProject,
			bkp,
			backupSCNames,
			backupSCAdoResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupServiceConnectionsCmd)

	backupServiceConnectionsCmd.Flags().StringVar(&backupSCSourceOrg, "source-org", "", "Source organization")
	backupServiceConnectionsCmd.Flags().StringVar(&backupSCSourceProject, "source-project", "", "Source project")
	backupServiceConnectionsCmd.Flags().StringSliceVar(&backupSCNames, "connections", []string{}, "Service connection names or 'all'")
	backupServiceConnectionsCmd.Flags().StringVar(&backupSCAdoResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	backupServiceConnectionsCmd.MarkFlagRequired("source-org")
	backupServiceConnectionsCmd.MarkFlagRequired("source-project")
	backupServiceConnectionsCmd.MarkFlagRequired("connections")
	backupServiceConnectionsCmd.MarkFlagRequired("ado-resource-guid")
}
