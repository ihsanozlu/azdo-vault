package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreVarSourceOrg string
var restoreVarSourceProject string
var restoreVarTargetOrg string
var restoreVarTargetProject string
var restoreVarGroups []string

var createVariableGroupsCmd = &cobra.Command{
	Use:   "create-variable-groups",
	Short: "Restore variable groups from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreVarSourceOrg)
		if err != nil {
			return err
		}

		if restoreVarTargetOrg == "" {
			restoreVarTargetOrg = restoreVarSourceOrg
		}

		if restoreVarTargetProject == "" {
			restoreVarTargetProject = restoreVarSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(restoreVarTargetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreVarSourceProject, "variable-groups")

		return internal.RestoreVariableGroupsFromBackup(
			targetOrgCfg.URL,
			restoreVarTargetProject,
			bkp,
			restoreVarGroups,
		)
	},
}

func init() {

	rootCmd.AddCommand(createVariableGroupsCmd)

	createVariableGroupsCmd.Flags().StringSliceVar(
		&restoreVarGroups,
		"groups",
		[]string{},
		"Variable group names or 'all'",
	)

	createVariableGroupsCmd.Flags().StringVar(
		&restoreVarSourceOrg,
		"source-org",
		"",
		"Source organization (where backup exists)",
	)

	createVariableGroupsCmd.Flags().StringVar(
		&restoreVarSourceProject,
		"source-project",
		"",
		"Source project (where backup exists)",
	)

	createVariableGroupsCmd.Flags().StringVar(
		&restoreVarTargetOrg,
		"target-org",
		"",
		"Target organization",
	)

	createVariableGroupsCmd.Flags().StringVar(
		&restoreVarTargetProject,
		"target-project",
		"",
		"Target project",
	)

	createVariableGroupsCmd.MarkFlagRequired("groups")
	createVariableGroupsCmd.MarkFlagRequired("source-org")
	createVariableGroupsCmd.MarkFlagRequired("source-project")
}
