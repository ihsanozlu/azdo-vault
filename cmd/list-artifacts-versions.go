package cmd

import (
	"fmt"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var artifactsVersionsOrg string
var artifactsVersionsProject string
var artifactsVersionsFeedID string
var artifactsPackageID string
var artifactsVersionsResourceGUID string

var listArtifactsVersionsCmd = &cobra.Command{
	Use:   "list-artifacts-versions",
	Short: "List versions of a package in an Azure Artifacts feed",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		_, orgCfg, err := cfg.ResolveOrganizationWithName(artifactsVersionsOrg)
		if err != nil {
			return err
		}

		versions, err := internal.ListPackageVersions(orgCfg.URL, artifactsVersionsProject, artifactsVersionsFeedID, artifactsPackageID, artifactsVersionsResourceGUID)
		if err != nil {
			return err
		}

		for _, v := range versions {
			fmt.Printf("- %s  (id=%s, listed=%t)\n", v.Version, v.ID, v.IsListed)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listArtifactsVersionsCmd)

	listArtifactsVersionsCmd.Flags().StringVar(&artifactsVersionsOrg, "org", "", "Organization name from config")
	listArtifactsVersionsCmd.Flags().StringVar(&artifactsVersionsProject, "project", "", "Project name")
	listArtifactsVersionsCmd.Flags().StringVar(&artifactsVersionsFeedID, "feed-id", "", "Feed ID (GUID)")
	listArtifactsVersionsCmd.Flags().StringVar(&artifactsPackageID, "package-id", "", "Package ID (GUID)")
	listArtifactsVersionsCmd.Flags().StringVar(&artifactsVersionsResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	listArtifactsVersionsCmd.MarkFlagRequired("org")
	listArtifactsVersionsCmd.MarkFlagRequired("project")
	listArtifactsVersionsCmd.MarkFlagRequired("feed-id")
	listArtifactsVersionsCmd.MarkFlagRequired("package-id")
	listArtifactsVersionsCmd.MarkFlagRequired("ado-resource-guid")
}
