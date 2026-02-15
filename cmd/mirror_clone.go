package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var project string
var repos []string
var orgName string

var mirrorCloneCmd = &cobra.Command{
	Use:   "mirror-clone",
	Short: "Mirror clone Azure DevOps repositories",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}
		orgCfg, err := cfg.ResolveOrganization(orgName)
		if err != nil {
			return err
		}

		os.Setenv("AZURE_DEVOPS_EXT_ORG_SERVICE_URL", orgCfg.URL)
		fmt.Printf("Organization URL: %s\n", orgCfg.URL)

		allRepos, err := internal.ListRepos(orgCfg.URL, project)
		if err != nil {
			return err
		}

		var selected []internal.Repo

		if len(repos) == 1 && strings.ToLower(repos[0]) == "all" {
			selected = allRepos
		} else {

			wanted := make(map[string]bool)
			for _, name := range repos {
				wanted[strings.ToLower(name)] = true
			}

			for _, r := range allRepos {
				if wanted[strings.ToLower(r.Name)] {
					selected = append(selected, r)
				}
			}
		}
		if len(selected) == 0 {
			fmt.Println("\nNo repositories matched your input.")

			if len(allRepos) == 0 {
				fmt.Println("No repositories found in this project.")
				return nil
			}

			fmt.Println("\nAvailable repositories in project", project, ":")
			for _, r := range allRepos {
				fmt.Println(" -", r.Name)
			}

			return fmt.Errorf("please check repository name or use --repos all")
		}

		fmt.Println("Repositories to be cloned:")
		for _, r := range selected {
			fmt.Println(" -", r.Name)
		}

		fmt.Print("Proceed? (y/N): ")
		in := bufio.NewReader(os.Stdin)
		resp, _ := in.ReadString('\n')
		if strings.ToLower(strings.TrimSpace(resp)) != "y" {
			fmt.Println("Aborted")
			return nil
		}

		for _, r := range selected {
			dest := filepath.Join(orgCfg.BackupRoot, orgNameOrDefault(cfg, orgName), project, "repos", r.Name+".git")
			fmt.Println("Cloning:", r.Name)
			if err := internal.MirrorClone(r.RemoteURL, dest); err != nil {
				return err
			}
		}

		fmt.Println("✔ Mirror clone completed\n\nThe path is:", orgCfg.BackupRoot)
		return nil
	},
}

func orgNameOrDefault(cfg *internal.Config, name string) string {
	if name != "" {
		return name
	}
	return cfg.DefaultOrganization
}

func init() {
	rootCmd.AddCommand(mirrorCloneCmd)
	mirrorCloneCmd.Flags().StringVar(&project, "project", "", "Azure DevOps project name")
	mirrorCloneCmd.Flags().StringSliceVar(&repos, "repos", []string{}, "Repo names or 'all'")
	mirrorCloneCmd.Flags().StringVar(&orgName, "org", "", "Organization name (optional)")
	mirrorCloneCmd.MarkFlagRequired("project")
	mirrorCloneCmd.MarkFlagRequired("repos")
}
