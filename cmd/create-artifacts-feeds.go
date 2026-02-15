package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreArtSourceOrg string
var restoreArtSourceProject string
var restoreArtTargetOrg string
var restoreArtTargetProject string
var restoreArtSelected []string
var restoreArtResourceGUID string

var createArtifactsFeedsCmd = &cobra.Command{
	Use:   "create-artifacts-feeds",
	Short: "Create Azure Artifacts feeds in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreArtSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := restoreArtTargetOrg
		if targetOrg == "" {
			targetOrg = restoreArtSourceOrg
		}
		targetProject := restoreArtTargetProject
		if targetProject == "" {
			targetProject = restoreArtSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreArtSourceProject, "artifacts/feeds")

		return internal.RestoreArtifactsFeedsFromBackup(
			sourceOrgCfg.URL,
			restoreArtSourceProject,
			targetOrgCfg.URL,
			targetProject,
			bkp,
			restoreArtSelected,
			restoreArtResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(createArtifactsFeedsCmd)

	createArtifactsFeedsCmd.Flags().StringVar(&restoreArtSourceOrg, "source-org", "", "Source organization (where backup exists)")
	createArtifactsFeedsCmd.Flags().StringVar(&restoreArtSourceProject, "source-project", "", "Source project (where backup exists)")
	createArtifactsFeedsCmd.Flags().StringVar(&restoreArtTargetOrg, "target-org", "", "Target organization (defaults to source-org)")
	createArtifactsFeedsCmd.Flags().StringVar(&restoreArtTargetProject, "target-project", "", "Target project (defaults to source-project)")
	createArtifactsFeedsCmd.Flags().StringSliceVar(&restoreArtSelected, "feeds", []string{"all"}, "Feed filenames or 'all'")
	createArtifactsFeedsCmd.Flags().StringVar(&restoreArtResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	createArtifactsFeedsCmd.MarkFlagRequired("source-org")
	createArtifactsFeedsCmd.MarkFlagRequired("source-project")
	createArtifactsFeedsCmd.MarkFlagRequired("ado-resource-guid")
}
