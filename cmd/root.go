package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "goscoop",
	Short:         "goscoop - Native Go rewrite of Scoop package manager",
	SilenceErrors: true,
	SilenceUsage:  true,
	Long: `goscoop is a fast native Go CLI that replaces Scoop's PowerShell backend.

It is compatible with Scoop buckets and manifest format,
providing faster installs and beautiful animated output.`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
