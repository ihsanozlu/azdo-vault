package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupTGSourceOrg string
var backupTGSourceProject string
var backupTGroups []string
var backupAdoResourceGUID string

var backupTaskGroupsCmd = &cobra.Command{
	Use:   "backup-task-groups",
	Short: "Backup task groups locally (JSON) from Azure DevOps",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(backupTGSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, backupTGSourceProject, "task-groups")

		return internal.BackupTaskGroups(
			sourceOrgCfg.URL,
			backupTGSourceProject,
			bkp,
			backupTGroups,
			backupAdoResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupTaskGroupsCmd)

	backupTaskGroupsCmd.Flags().StringVar(
		&backupTGSourceOrg,
		"source-org",
		"",
		"Source organization",
	)

	backupTaskGroupsCmd.Flags().StringVar(
		&backupTGSourceProject,
		"source-project",
		"",
		"Source project",
	)

	backupTaskGroupsCmd.Flags().StringSliceVar(
		&backupTGroups,
		"groups",
		[]string{},
		"Task group names or 'all'",
	)

	backupTaskGroupsCmd.Flags().StringVar(
		&backupAdoResourceGUID,
		"ado-resource-guid",
		"",
		"Azure DevOps AAD resource GUID (required for az rest)",
	)

	backupTaskGroupsCmd.MarkFlagRequired("source-org")
	backupTaskGroupsCmd.MarkFlagRequired("source-project")
	backupTaskGroupsCmd.MarkFlagRequired("groups")
	backupTaskGroupsCmd.MarkFlagRequired("ado-resource-guid")
}
