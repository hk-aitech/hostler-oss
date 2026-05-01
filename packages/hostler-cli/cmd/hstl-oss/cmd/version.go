package cmd

import (
	"fmt"

	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		if outputFormat == "json" {
			Out.Print(map[string]string{
				"version": Version,
				"commit":  Commit,
				"date":    Date,
			})
		} else {
			fmt.Printf("hstl %s (commit: %s, built: %s)\n", Version, Commit, Date)
		}
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
