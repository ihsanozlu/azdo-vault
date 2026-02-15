package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restoreRelSourceOrg string
var restoreRelSourceProject string
var restoreRelTargetOrg string
var restoreRelTargetProject string
var restoreRelDefinitions []string
var restoreRelAdoResourceGUID string
var restoreRelQueueMap []string
var restoreRelDefaultQueue string

var createReleaseDefinitionsCmd = &cobra.Command{
	Use:   "create-release-definitions",
	Short: "Create classic release definitions in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restoreRelSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := restoreRelTargetOrg
		if targetOrg == "" {
			targetOrg = restoreRelSourceOrg
		}
		targetProject := restoreRelTargetProject
		if targetProject == "" {
			targetProject = restoreRelSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restoreRelSourceProject, "release-definitions")

		return internal.RestoreReleaseDefinitionsFromBackup(
			sourceOrgCfg.URL,
			restoreRelSourceProject,
			targetOrgCfg.URL,
			targetProject,
			bkp,
			restoreRelDefinitions,
			restoreRelAdoResourceGUID,
			restoreRelQueueMap,
			restoreRelDefaultQueue,
		)

	},
}

func init() {
	rootCmd.AddCommand(createReleaseDefinitionsCmd)

	createReleaseDefinitionsCmd.Flags().StringSliceVar(&restoreRelDefinitions, "definitions", []string{}, "Release definition names or 'all'")
	createReleaseDefinitionsCmd.Flags().StringVar(&restoreRelSourceOrg, "source-org", "", "Source organization (where backup exists)")
	createReleaseDefinitionsCmd.Flags().StringVar(&restoreRelSourceProject, "source-project", "", "Source project (where backup exists)")
	createReleaseDefinitionsCmd.Flags().StringVar(&restoreRelTargetOrg, "target-org", "", "Target organization (defaults to source-org if omitted)")
	createReleaseDefinitionsCmd.Flags().StringVar(&restoreRelTargetProject, "target-project", "", "Target project (defaults to source-project if omitted)")
	createReleaseDefinitionsCmd.Flags().StringVar(&restoreRelAdoResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")
	createReleaseDefinitionsCmd.Flags().StringSliceVar(&restoreRelQueueMap, "queue-map", []string{}, "Queue mapping in form 'SourceQueue=TargetQueue' (repeatable)")
	createReleaseDefinitionsCmd.Flags().StringVar(&restoreRelDefaultQueue, "default-queue", "", "Fallback target queue name when no mapping/match exists")

	createReleaseDefinitionsCmd.MarkFlagRequired("definitions")
	createReleaseDefinitionsCmd.MarkFlagRequired("source-org")
	createReleaseDefinitionsCmd.MarkFlagRequired("source-project")
	createReleaseDefinitionsCmd.MarkFlagRequired("ado-resource-guid")
}
