package cmd

import (
	"fmt"
	"os"

	"azdo-vault/internal"

	"github.com/spf13/cobra"
)

var configureCmd = &cobra.Command{
	Use:   "configure",
	Short: "Manage Azure DevOps organizations",
}

var addOrgName string
var addOrgUrl string

var configureAddCmd = &cobra.Command{
	Use:   "add",
	Short: "Add an Azure DevOps {organization} short name (not the full URL) : https://dev.azure.com/{organization}",
	RunE: func(cmd *cobra.Command, args []string) error {
		if addOrgName == "" || addOrgUrl == "" {
			return fmt.Errorf("name and organization are required")
		}

		var cfg *internal.Config

		// Try loading config
		loadedCfg, err := internal.LoadConfig()
		if err != nil {
			// If config file doesn't exist → initialize new config
			if os.IsNotExist(err) {
				cfg = &internal.Config{
					Organizations: make(map[string]internal.OrganizationConfig),
				}
			} else {
				return err
			}
		} else {
			cfg = loadedCfg
		}

		// Ensure Organizations map is initialized
		if cfg.Organizations == nil {
			cfg.Organizations = make(map[string]internal.OrganizationConfig)
		}

		home, _ := os.UserHomeDir()

		cfg.Organizations[addOrgName] = internal.OrganizationConfig{
			URL:        "https://dev.azure.com/" + addOrgUrl,
			BackupRoot: home + "/azdo-vaults",
		}

		// If first organization → set as default
		if cfg.DefaultOrganization == "" {
			cfg.DefaultOrganization = addOrgName
			fmt.Println("✔ First organization added. Set as default:", addOrgName)
		} else {
			fmt.Println("✔ Organization added:", addOrgName)
		}

		return internal.SaveConfig(cfg)
	},
}

var configureDefaultCmd = &cobra.Command{
	Use:   "default [name]",
	Short: "Set default organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := internal.LoadConfig()
		if err != nil {
			return err
		}

		if _, ok := cfg.Organizations[args[0]]; !ok {
			return fmt.Errorf("organization '%s' not found", args[0])
		}

		cfg.DefaultOrganization = args[0]
		if err := internal.SaveConfig(cfg); err != nil {
			return err
		}

		fmt.Println("✔ Default organization set to:", args[0])
		return nil
	},
}

var configureListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured organizations",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := internal.LoadConfig()
		if err != nil {
			return err
		}

		for name := range cfg.Organizations {
			if name == cfg.DefaultOrganization {
				fmt.Println("*", name, "(default)")
			} else {
				fmt.Println(" ", name)
			}
		}
		return nil
	},
}

var configureShowCmd = &cobra.Command{
	Use:   "show",
	Short: "Show current configuration",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := internal.LoadConfig()
		if err != nil {
			if os.IsNotExist(err) {
				fmt.Println("No configuration found.")
				return nil
			}
			return err
		}

		fmt.Println("Default Organization:", cfg.DefaultOrganization)
		fmt.Println()
		fmt.Println("Organizations:")

		for name, org := range cfg.Organizations {
			prefix := " "
			if name == cfg.DefaultOrganization {
				prefix = "*"
			}

			fmt.Printf(" %s %s\n", prefix, name)
			fmt.Printf("   URL: %s\n", org.URL)
			fmt.Printf("   BackupRoot: %s\n\n", org.BackupRoot)
		}

		return nil
	},
}

var configureRemoveCmd = &cobra.Command{
	Use:   "remove [name]",
	Short: "Remove an organization",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		orgName := args[0]

		cfg, err := internal.LoadConfig()
		if err != nil {
			return fmt.Errorf("no configuration found")
		}

		if _, exists := cfg.Organizations[orgName]; !exists {
			return fmt.Errorf("organization '%s' not found", orgName)
		}

		delete(cfg.Organizations, orgName)
		fmt.Println("Removed organization:", orgName)

		// If removed org was default → assign new default
		if cfg.DefaultOrganization == orgName {
			if len(cfg.Organizations) == 0 {
				cfg.DefaultOrganization = ""
				fmt.Println("No organizations remaining. Default cleared.")
			} else {
				// Pick first available org as new default
				for name := range cfg.Organizations {
					cfg.DefaultOrganization = name
					fmt.Println("New default organization set to:", name)
					break
				}
			}
		}

		return internal.SaveConfig(cfg)
	},
}

func init() {
	rootCmd.AddCommand(configureCmd)

	configureCmd.AddCommand(configureAddCmd)
	configureCmd.AddCommand(configureShowCmd)
	configureCmd.AddCommand(configureRemoveCmd)
	configureCmd.AddCommand(configureDefaultCmd)
	configureCmd.AddCommand(configureListCmd)

	configureAddCmd.Flags().StringVar(&addOrgName, "name", "", "Organization alias")
	configureAddCmd.Flags().StringVar(&addOrgUrl, "org", "", "Azure DevOps organization short name (not the full URL)")
}
