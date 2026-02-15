package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var restorePolSourceOrg string
var restorePolSourceProject string
var restorePolTargetOrg string
var restorePolTargetProject string
var restorePolSelected []string
var restorePolResourceGUID string

var createBranchPoliciesCmd = &cobra.Command{
	Use:   "create-branch-policies",
	Short: "Create branch policies in target org/project from local backup",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(restorePolSourceOrg)
		if err != nil {
			return err
		}

		targetOrg := restorePolTargetOrg
		if targetOrg == "" {
			targetOrg = restorePolSourceOrg
		}
		targetProject := restorePolTargetProject
		if targetProject == "" {
			targetProject = restorePolSourceProject
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, restorePolSourceProject, "branch-policies")

		return internal.RestoreBranchPoliciesFromBackup(
			sourceOrgCfg.URL,
			restorePolSourceProject,
			targetOrgCfg.URL,
			targetProject,
			bkp,
			restorePolSelected,
			restorePolResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(createBranchPoliciesCmd)

	createBranchPoliciesCmd.Flags().StringVar(&restorePolSourceOrg, "source-org", "", "Source organization (where backup exists)")
	createBranchPoliciesCmd.Flags().StringVar(&restorePolSourceProject, "source-project", "", "Source project (where backup exists)")
	createBranchPoliciesCmd.Flags().StringVar(&restorePolTargetOrg, "target-org", "", "Target organization (defaults to source-org if omitted)")
	createBranchPoliciesCmd.Flags().StringVar(&restorePolTargetProject, "target-project", "", "Target project (defaults to source-project if omitted)")
	createBranchPoliciesCmd.Flags().StringSliceVar(&restorePolSelected, "policies", []string{"all"}, "Policy filenames, policy ids, or 'all'")
	createBranchPoliciesCmd.Flags().StringVar(&restorePolResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	createBranchPoliciesCmd.MarkFlagRequired("source-org")
	createBranchPoliciesCmd.MarkFlagRequired("source-project")
	createBranchPoliciesCmd.MarkFlagRequired("ado-resource-guid")
}
