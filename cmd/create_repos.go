package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var createRepos []string
var sourceProject string
var targetProject string
var sourceOrg string
var targetOrg string

var createReposCmd = &cobra.Command{
	Use:   "create-repos",
	Short: "Create repositories in Azure DevOps",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(sourceOrg)
		if err != nil {
			return err
		}

		if targetOrg == "" {
			targetOrg = sourceOrg
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(targetOrg)
		if err != nil {
			return err
		}
		if targetProject == "" {
			targetProject = sourceProject
		}

		var repoNames []string

		// If "all" → read from backup directory
		if len(createRepos) == 1 && createRepos[0] == "all" {

			repoPath := filepath.Join(
				sourceOrgCfg.BackupRoot,
				sourceOrgName,
				sourceProject,
				"repos",
			)

			files, err := os.ReadDir(repoPath)
			if err != nil {
				return fmt.Errorf("failed reading backup repos: %w", err)
			}

			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".git") {
					repoNames = append(repoNames,
						strings.TrimSuffix(f.Name(), ".git"))
				}
			}

		} else {
			repoNames = createRepos
		}

		if len(repoNames) == 0 {
			return fmt.Errorf("no repositories found to create")
		}

		fmt.Println("Repositories to create:")
		for _, r := range repoNames {
			fmt.Println(" -", r)
		}

		for _, repo := range repoNames {

			exists, _ := internal.RepoExists(targetOrgCfg.URL, targetProject, repo)
			if exists {
				fmt.Println("✔ Already exists:", repo)
				continue
			}

			fmt.Println("Creating:", repo)
			if err := internal.CreateRepo(targetOrgCfg.URL, targetProject, repo); err != nil {
				return fmt.Errorf("Failed creating repo %s: %w \n\nBefore everyting, ensure you have permissions and the project exists.", repo, err)
			}
		}

		fmt.Println("✔ Repository creation completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(createReposCmd)

	createReposCmd.Flags().StringSliceVar(
		&createRepos,
		"repos",
		[]string{},
		"Repository names or 'all'",
	)

	createReposCmd.Flags().StringVar(
		&sourceProject,
		"source-project",
		"",
		"Source project name (where backup exists)",
	)

	createReposCmd.Flags().StringVar(
		&targetProject,
		"target-project",
		"",
		"Target project name (where repos will be created)",
	)

	createReposCmd.Flags().StringVar(
		&sourceOrg,
		"source-org",
		"",
		"Source organization name (where backup exists)",
	)

	createReposCmd.Flags().StringVar(
		&targetOrg,
		"target-org",
		"",
		"Target organization name (where repos will be created)",
	)
	createReposCmd.MarkFlagRequired("repos")
	createReposCmd.MarkFlagRequired("source-project")

}
