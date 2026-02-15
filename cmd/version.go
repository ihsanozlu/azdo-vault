package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	Version   = "dev"
	Commit    = "none"
	BuildDate = "unknown"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print azdo-vault version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("azdo-vault")
		fmt.Println(" Version   :", Version)
		fmt.Println(" Commit    :", Commit)
		fmt.Println(" BuildDate :", BuildDate)
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
