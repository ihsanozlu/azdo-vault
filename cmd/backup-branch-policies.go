package cmd

import (
	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var backupPolSourceOrg string
var backupPolSourceProject string
var backupPolRepos []string
var backupPolResourceGUID string

var backupBranchPoliciesCmd = &cobra.Command{
	Use:   "backup-branch-policies",
	Short: "Backup branch policies locally (JSON) from Azure DevOps",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(backupPolSourceOrg)
		if err != nil {
			return err
		}

		bkp := backupPath(cfg, sourceOrgName, sourceOrgCfg, backupPolSourceProject, "branch-policies")

		return internal.BackupBranchPolicies(
			sourceOrgCfg.URL,
			backupPolSourceProject,
			bkp,
			backupPolRepos,
			backupPolResourceGUID,
		)
	},
}

func init() {
	rootCmd.AddCommand(backupBranchPoliciesCmd)

	backupBranchPoliciesCmd.Flags().StringVar(&backupPolSourceOrg, "source-org", "", "Source organization")
	backupBranchPoliciesCmd.Flags().StringVar(&backupPolSourceProject, "source-project", "", "Source project")
	backupBranchPoliciesCmd.Flags().StringSliceVar(&backupPolRepos, "repos", []string{"all"}, "Repo names or 'all' (filters policies by scope.repositoryId, includes repoId=null policies too)")
	backupBranchPoliciesCmd.Flags().StringVar(&backupPolResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	backupBranchPoliciesCmd.MarkFlagRequired("source-org")
	backupBranchPoliciesCmd.MarkFlagRequired("source-project")
	backupBranchPoliciesCmd.MarkFlagRequired("ado-resource-guid")
}
