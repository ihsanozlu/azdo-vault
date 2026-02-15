package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupWikisSourceOrg string
var backupWikisSourceProject string
var backupWikisSelected []string
var backupWikisResourceGUID string

var backupWikisCmd = &cobra.Command{
	Use:   "backup-wikis",
	Short: "Backup Azure DevOps wikis (metadata + project wiki git repos)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(backupWikisSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, backupWikisSourceProject, "wikis")

		return internal.BackupWikis(
			sourceOrgCfg.URL,
			backupWikisSourceProject,
			bkp,
			backupWikisSelected,
			backupWikisResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupWikisCmd)

	backupWikisCmd.Flags().StringVar(&backupWikisSourceOrg, "source-org", "", "Source organization")
	backupWikisCmd.Flags().StringVar(&backupWikisSourceProject, "source-project", "", "Source project")
	backupWikisCmd.Flags().StringSliceVar(&backupWikisSelected, "wikis", []string{"all"}, "Wiki names/ids or 'all'")
	backupWikisCmd.Flags().StringVar(&backupWikisResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	backupWikisCmd.MarkFlagRequired("source-org")
	backupWikisCmd.MarkFlagRequired("source-project")
	backupWikisCmd.MarkFlagRequired("ado-resource-guid")
}
