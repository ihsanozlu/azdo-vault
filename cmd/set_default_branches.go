package cmd

import (
	"fmt"
	"strings"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var setDefOrg string
var setDefProject string
var setDefBranch string
var setDefRepos []string
var setDefResourceGUID string

var setDefaultBranchesCmd = &cobra.Command{
	Use:   "set-default-branches",
	Short: "Set default branch for repos in a project (e.g., development)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		orgName, orgCfg, err := cfg.ResolveOrganizationWithName(setDefOrg)
		if err != nil {
			return err
		}
		_ = orgName // not required; kept for parity

		repos, err := internal.ListRepos(orgCfg.URL, setDefProject)
		if err != nil {
			return fmt.Errorf("failed to list repos: %w", err)
		}
		if len(repos) == 0 {
			fmt.Println("No repos found")
			return nil
		}

		filter := map[string]bool{}
		if len(setDefRepos) == 1 && strings.EqualFold(setDefRepos[0], "all") {
			// no filter
		} else {
			for _, r := range setDefRepos {
				filter[strings.ToLower(strings.TrimSpace(r))] = true
			}
		}

		changed := 0
		for _, r := range repos {
			if len(filter) > 0 && !filter[strings.ToLower(r.Name)] {
				continue
			}

			if err := internal.UpdateRepoDefaultBranch(orgCfg.URL, setDefProject, r.Id, setDefBranch, setDefResourceGUID); err != nil {
				fmt.Printf("⚠ Failed: repo=%s id=%s err=%v\n", r.Name, r.Id, err)
				continue
			}
			fmt.Printf("✔ Updated defaultBranch: repo=%s -> %s\n", r.Name, setDefBranch)
			changed++
		}

		fmt.Printf("Done. Updated %d repos.\n", changed)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(setDefaultBranchesCmd)

	setDefaultBranchesCmd.Flags().StringVar(&setDefOrg, "org", "", "Organization name from config")
	setDefaultBranchesCmd.Flags().StringVar(&setDefProject, "project", "", "Project name")
	setDefaultBranchesCmd.Flags().StringVar(&setDefBranch, "branch", "refs/heads/development", "Default branch (e.g. refs/heads/development or development)")
	setDefaultBranchesCmd.Flags().StringSliceVar(&setDefRepos, "repos", []string{"all"}, "Repo names or 'all'")
	setDefaultBranchesCmd.Flags().StringVar(&setDefResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	setDefaultBranchesCmd.MarkFlagRequired("org")
	setDefaultBranchesCmd.MarkFlagRequired("project")
	setDefaultBranchesCmd.MarkFlagRequired("ado-resource-guid")
}
