package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreSCSourceOrg string
var restoreSCSourceProject string
var restoreSCTargetOrg string
var restoreSCTargetProject string
var restoreSCNames []string
var restoreSCAdoResourceGUID string

var createServiceConnectionsCmd = &cobra.Command{
	Use:   "create-service-connections",
	Short: "Create service connections in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreSCSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := restoreSCTargetOrg
		if targetOrg == "" {
			targetOrg = restoreSCSourceOrg
		}
		targetProject := restoreSCTargetProject
		if targetProject == "" {
			targetProject = restoreSCSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreSCSourceProject, "service-connections")

		return internal.RestoreServiceConnectionsFromBackup(
			targetOrgCfg.URL,
			targetProject,
			bkp,
			restoreSCNames,
			restoreSCAdoResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(createServiceConnectionsCmd)

	createServiceConnectionsCmd.Flags().StringVar(&restoreSCSourceOrg, "source-org", "", "Source organization (where backup exists)")
	createServiceConnectionsCmd.Flags().StringVar(&restoreSCSourceProject, "source-project", "", "Source project (where backup exists)")
	createServiceConnectionsCmd.Flags().StringVar(&restoreSCTargetOrg, "target-org", "", "Target organization (defaults to source-org if omitted)")
	createServiceConnectionsCmd.Flags().StringVar(&restoreSCTargetProject, "target-project", "", "Target project (defaults to source-project if omitted)")
	createServiceConnectionsCmd.Flags().StringSliceVar(&restoreSCNames, "connections", []string{}, "Service connection names or 'all'")
	createServiceConnectionsCmd.Flags().StringVar(&restoreSCAdoResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	createServiceConnectionsCmd.MarkFlagRequired("source-org")
	createServiceConnectionsCmd.MarkFlagRequired("source-project")
	createServiceConnectionsCmd.MarkFlagRequired("connections")
	createServiceConnectionsCmd.MarkFlagRequired("ado-resource-guid")
}
