package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupArtSourceOrg string
var backupArtSourceProject string
var backupArtResourceGUID string

var backupArtifactsFeedsCmd = &cobra.Command{
	Use:   "backup-artifacts-feeds",
	Short: "Backup Azure Artifacts feeds locally (JSON)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(backupArtSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, backupArtSourceProject, "artifacts/feeds")

		return internal.BackupArtifactsFeeds(
			sourceOrgCfg.URL,
			backupArtSourceProject,
			bkp,
			backupArtResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupArtifactsFeedsCmd)

	backupArtifactsFeedsCmd.Flags().StringVar(&backupArtSourceOrg, "source-org", "", "Source organization")
	backupArtifactsFeedsCmd.Flags().StringVar(&backupArtSourceProject, "source-project", "", "Source project")
	backupArtifactsFeedsCmd.Flags().StringVar(&backupArtResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	backupArtifactsFeedsCmd.MarkFlagRequired("source-org")
	backupArtifactsFeedsCmd.MarkFlagRequired("source-project")
	backupArtifactsFeedsCmd.MarkFlagRequired("ado-resource-guid")
}
