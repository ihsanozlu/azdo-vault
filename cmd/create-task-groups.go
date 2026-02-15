package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreTGSourceOrg string
var restoreTGSourceProject string
var restoreTGTargetOrg string
var restoreTGTargetProject string
var restoreTGroups []string
var restoreAdoResourceGUID string

var createTaskGroupsCmd = &cobra.Command{
	Use:   "create-task-groups",
	Short: "Create task groups in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreTGSourceOrg)

		if err != nil {
			return err
		}

		// Default target values
		resolvedTargetOrg := restoreTGTargetOrg
		if resolvedTargetOrg == "" {
			resolvedTargetOrg = restoreTGSourceOrg
		}

		resolvedTargetProject := restoreTGTargetProject
		if resolvedTargetProject == "" {
			resolvedTargetProject = restoreTGSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(resolvedTargetOrg)
		if err != nil {
			return err
		}

		// backupPath := filepath.Join(
		// 	sourceOrgCfg.BackupRoot,
		// 	sourceOrgName,
		// 	restoreTGSourceProject,
		// 	"task-groups",
		// )
		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreTGSourceProject, "task-groups")

		// ✅ NEW: include source org URL + source project (for endpoint ID -> name -> target ID remap)
		return internal.RestoreTaskGroupsFromBackup(
			sourceOrgCfg.URL,
			restoreTGSourceProject,
			targetOrgCfg.URL,
			resolvedTargetProject,
			bkp,
			restoreTGroups,
			restoreAdoResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(createTaskGroupsCmd)

	createTaskGroupsCmd.Flags().StringSliceVar(&restoreTGroups, "groups", []string{}, "Task group names or 'all'")
	createTaskGroupsCmd.Flags().StringVar(&restoreTGSourceOrg, "source-org", "", "Source organization (where backup exists)")
	createTaskGroupsCmd.Flags().StringVar(&restoreTGSourceProject, "source-project", "", "Source project (where backup exists)")
	createTaskGroupsCmd.Flags().StringVar(&restoreTGTargetOrg, "target-org", "", "Target organization (defaults to source-org if omitted)")
	createTaskGroupsCmd.Flags().StringVar(&restoreTGTargetProject, "target-project", "", "Target project (defaults to source-project if omitted)")
	createTaskGroupsCmd.Flags().StringVar(&restoreAdoResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	createTaskGroupsCmd.MarkFlagRequired("groups")
	createTaskGroupsCmd.MarkFlagRequired("source-org")
	createTaskGroupsCmd.MarkFlagRequired("source-project")
	createTaskGroupsCmd.MarkFlagRequired("ado-resource-guid")
}
