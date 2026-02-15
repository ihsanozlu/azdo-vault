package cmd

import (
	"fmt"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var artifactsOrg string
var artifactsProject string
var artifactsResourceGUID string

var listArtifactsFeedsCmd = &cobra.Command{
	Use:   "list-artifacts-feeds",
	Short: "List Azure Artifacts feeds in a project",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		orgName, orgCfg, err := cfg.ResolveOrganizationWithName(artifactsOrg)
		if err != nil {
			return err
		}
		_ = orgName

		feeds, err := internal.ListFeeds(orgCfg.URL, artifactsProject, artifactsResourceGUID)
		if err != nil {
			return err
		}

		for _, f := range feeds {
			fmt.Printf("- %s  (id=%s)\n", f.Name, f.ID)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listArtifactsFeedsCmd)

	listArtifactsFeedsCmd.Flags().StringVar(&artifactsOrg, "org", "", "Organization name from config (e.g. dot)")
	listArtifactsFeedsCmd.Flags().StringVar(&artifactsProject, "project", "", "Project name")
	listArtifactsFeedsCmd.Flags().StringVar(&artifactsResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	listArtifactsFeedsCmd.MarkFlagRequired("org")
	listArtifactsFeedsCmd.MarkFlagRequired("project")
	listArtifactsFeedsCmd.MarkFlagRequired("ado-resource-guid")
}
