package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var pushRepos []string

// var pushProject string
// var pushOrg string
var pushSourceProject string
var pushTargetProject string
var pushSourceOrg string
var pushTargetOrg string

var pushAllAndTagsCmd = &cobra.Command{
	Use:   "push-all-and-tags",
	Short: "Push all branches and tags to Azure DevOps repositories",
	RunE: func(cmd *cobra.Command, args []string) error {

		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}
		sourceOrgName, sourceOrgCfg, err := cfg.ResolveOrganizationWithName(pushSourceOrg)
		if err != nil {
			return err
		}

		if pushTargetOrg == "" {
			pushTargetOrg = pushSourceOrg
		}

		_, targetOrgCfg, err := cfg.ResolveOrganizationWithName(pushTargetOrg)
		if err != nil {
			return err
		}

		if pushTargetProject == "" {
			pushTargetProject = pushSourceProject
		}

		repoBasePath := filepath.Join(
			sourceOrgCfg.BackupRoot,
			sourceOrgName,
			pushSourceProject,
			"repos",
		)

		var repoNames []string

		if len(pushRepos) == 1 && pushRepos[0] == "all" {

			files, err := os.ReadDir(repoBasePath)
			if err != nil {
				return err
			}

			for _, f := range files {
				if strings.HasSuffix(f.Name(), ".git") {
					repoNames = append(repoNames,
						strings.TrimSuffix(f.Name(), ".git"))
				}
			}

		} else {
			repoNames = pushRepos
		}

		if len(repoNames) == 0 {
			return fmt.Errorf("no repositories found to push")
		}

		for _, repo := range repoNames {

			localPath := filepath.Join(repoBasePath, repo+".git")

			remoteURL, err := internal.GetRepoRemoteURL(
				targetOrgCfg.URL,
				pushTargetProject,
				repo,
			)
			if err != nil {
				return err
			}

			fmt.Println("Pushing:", repo)

			if err := internal.PushAllAndTags(localPath, remoteURL); err != nil {
				return err
			}
		}

		fmt.Println("✔ Mirror push completed")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pushAllAndTagsCmd)

	pushAllAndTagsCmd.Flags().StringSliceVar(
		&pushRepos,
		"repos",
		[]string{},
		"Repository names or 'all'",
	)

	pushAllAndTagsCmd.Flags().StringVar(
		&pushSourceProject,
		"source-project",
		"",
		"Source project (where backup exists)",
	)

	pushAllAndTagsCmd.Flags().StringVar(
		&pushTargetProject,
		"target-project",
		"",
		"Target project (where repos will be pushed)",
	)

	pushAllAndTagsCmd.Flags().StringVar(
		&pushSourceOrg,
		"source-org",
		"",
		"Source organization (where backup exists)",
	)

	pushAllAndTagsCmd.Flags().StringVar(
		&pushTargetOrg,
		"target-org",
		"",
		"Target organization (where repos will be pushed)",
	)

	pushAllAndTagsCmd.MarkFlagRequired("repos")
	pushAllAndTagsCmd.MarkFlagRequired("source-project")
}
