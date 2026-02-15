package cmd

import (
	"fmt"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var artifactsPackagesOrg string
var artifactsPackagesProject string
var artifactsFeedID string
var artifactsProtocol string
var artifactsPackagesResourceGUID string

var listArtifactsPackagesCmd = &cobra.Command{
	Use:   "list-artifacts-packages",
	Short: "List packages in an Azure Artifacts feed (npm/maven/nuget/pypi...)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := mustLoadConfig()
		if err != nil {
			return err
		}

		_, orgCfg, err := cfg.ResolveOrganizationWithName(artifactsPackagesOrg)
		if err != nil {
			return err
		}

		pkgs, err := internal.ListPackages(orgCfg.URL, artifactsPackagesProject, artifactsFeedID, artifactsProtocol, artifactsPackagesResourceGUID)
		if err != nil {
			return err
		}

		for _, p := range pkgs {
			fmt.Printf("- %s  (id=%s, protocol=%s)\n", p.Name, p.ID, p.ProtocolType)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(listArtifactsPackagesCmd)

	listArtifactsPackagesCmd.Flags().StringVar(&artifactsPackagesOrg, "org", "", "Organization name from config")
	listArtifactsPackagesCmd.Flags().StringVar(&artifactsPackagesProject, "project", "", "Project name")
	listArtifactsPackagesCmd.Flags().StringVar(&artifactsFeedID, "feed-id", "", "Feed ID (GUID)")
	listArtifactsPackagesCmd.Flags().StringVar(&artifactsProtocol, "protocol", "", "Protocol filter: npm|maven|nuget|pypi (optional)")
	listArtifactsPackagesCmd.Flags().StringVar(&artifactsPackagesResourceGUID, "ado-resource-guid", "", "Azure DevOps AAD resource GUID (required for az rest)")

	listArtifactsPackagesCmd.MarkFlagRequired("org")
	listArtifactsPackagesCmd.MarkFlagRequired("project")
	listArtifactsPackagesCmd.MarkFlagRequired("feed-id")
	listArtifactsFeedsCmd.MarkFlagRequired("ado-resource-guid")
}
