package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreWikisSourceOrg string
var restoreWikisSourceProject string
var restoreWikisTargetOrg string
var restoreWikisTargetProject string
var restoreWikisSelected []string
var restoreWikisResourceGUID string

var createWikisCmd = &cobra.Command{
	Use:   "create-wikis",
	Short: "Restore Azure DevOps wikis into target org/project from backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreWikisSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := restoreWikisTargetOrg
		if targetOrg == "" {
			targetOrg = restoreWikisSourceOrg
		}
		targetProject := restoreWikisTargetProject
		if targetProject == "" {
			targetProject = restoreWikisSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreWikisSourceProject, "wikis")

		return internal.RestoreWikisFromBackup(
			sourceOrgCfg.URL,
			restoreWikisSourceProject,
			targetOrgCfg.URL,
			targetProject,
			bkp,
			restoreWikisSelected,
			restoreWikisResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(createWikisCmd)

	createWikisCmd.Flags().StringVar(&restoreWikisSourceOrg, "source-org", "", "Source org (where backup exists)")
	createWikisCmd.Flags().StringVar(&restoreWikisSourceProject, "source-project", "", "Source project (where backup exists)")
	createWikisCmd.Flags().StringVar(&restoreWikisTargetOrg, "target-org", "", "Target org (defaults to source-org)")
	createWikisCmd.Flags().StringVar(&restoreWikisTargetProject, "target-project", "", "Target project (defaults to source-project)")
	createWikisCmd.Flags().StringSliceVar(&restoreWikisSelected, "wikis", []string{"all"}, "Wiki names/ids or 'all'")
	createWikisCmd.Flags().StringVar(&restoreWikisResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	createWikisCmd.MarkFlagRequired("source-org")
	createWikisCmd.MarkFlagRequired("source-project")
	createWikisCmd.MarkFlagRequired("ado-resource-guid")
}
