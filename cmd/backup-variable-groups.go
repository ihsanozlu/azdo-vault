package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupVarGroups []string
var backupVarSourceOrg string
var backupVarSourceProject string

var backupVariableGroupsCmd = &cobra.Command{
	Use:   "backup-variable-groups",
	Short: "Backup variable groups locally",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		orgName, orgCfg, err := cfg.ResolveOrganizationWithName(backupVarSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, orgName, orgCfg, backupVarSourceProject, "variable-groups")

		return internal.BackupVariableGroups(
			orgCfg.URL,
			backupVarSourceProject,
			bkp,
			backupVarGroups,
		)
	},
}

func init() {

	rootCmd.AddCommand(backupVariableGroupsCmd)

	backupVariableGroupsCmd.Flags().StringVar(
		&backupVarSourceOrg,
		"source-org",
		"",
		"Source organization",
	)

	backupVariableGroupsCmd.Flags().StringVar(
		&backupVarSourceProject,
		"source-project",
		"",
		"Source project",
	)

	backupVariableGroupsCmd.Flags().StringSliceVar(
		&backupVarGroups,
		"groups",
		[]string{},
		"Variable group names or 'all'",
	)

	backupVariableGroupsCmd.MarkFlagRequired("source-org")
	backupVariableGroupsCmd.MarkFlagRequired("source-project")
	backupVariableGroupsCmd.MarkFlagRequired("groups")
}
