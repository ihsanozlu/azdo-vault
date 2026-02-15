package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var bldRestoreSourceOrg string
var bldRestoreSourceProject string
var bldRestoreTargetOrg string
var bldRestoreTargetProject string
var bldRestoreNames []string
var bldRestoreResourceGUID string
var bldRestoreQueueMap []string
var bldRestoreDefaultQueue string

var createBuildDefsCmd = &cobra.Command{
	Use:   "create-build-definitions",
	Short: "Create classic build definitions in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(bldRestoreSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := bldRestoreTargetOrg
		if targetOrg == "" {
			targetOrg = bldRestoreSourceOrg
		}
		targetProject := bldRestoreTargetProject
		if targetProject == "" {
			targetProject = bldRestoreSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, bldRestoreSourceProject, "build-definitions")

		return internal.RestoreBuildDefinitionsFromBackup(
			sourceOrgCfg.URL,
			bldRestoreSourceProject,
			targetOrgCfg.URL,
			targetProject,
			bkp,
			bldRestoreNames,
			bldRestoreResourceGUID,
			bldRestoreQueueMap,
			bldRestoreDefaultQueue,
		)
	},
}

func init() {
	rootCmd.AddCommand(createBuildDefsCmd)

	createBuildDefsCmd.Flags().StringVar(&bldRestoreSourceOrg, "source-org", "", "Source organization (where backup exists)")
	createBuildDefsCmd.Flags().StringVar(&bldRestoreSourceProject, "source-project", "", "Source project (where backup exists)")
	createBuildDefsCmd.Flags().StringVar(&bldRestoreTargetOrg, "target-org", "", "Target organization (defaults to source-org if omitted)")
	createBuildDefsCmd.Flags().StringVar(&bldRestoreTargetProject, "target-project", "", "Target project (defaults to source-project if omitted)")
	createBuildDefsCmd.Flags().StringSliceVar(&bldRestoreNames, "definitions", []string{}, "Build definition names or 'all'")
	createBuildDefsCmd.Flags().StringVar(&bldRestoreResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")
	createBuildDefsCmd.Flags().StringSliceVar(&bldRestoreQueueMap, "queue-map", []string{}, "Queue mapping in form 'SourceQueue=TargetQueue' (repeatable)")
	createBuildDefsCmd.Flags().StringVar(&bldRestoreDefaultQueue, "default-queue", "", "Fallback target queue name when no mapping/match exists")

	createBuildDefsCmd.MarkFlagRequired("source-org")
	createBuildDefsCmd.MarkFlagRequired("source-project")
	createBuildDefsCmd.MarkFlagRequired("definitions")
	createBuildDefsCmd.MarkFlagRequired("ado-resource-guid")
}
