package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile     string
	userLicense string

	rootCmd = &cobra.Command{
		Use:   "loupebox",
		Short: "Photo aggregation utility",
		Long:  `Loupebox aggregates your photos into a single folder organized by the date taken.`,
	}
)

// Execute executes the root command.
func Execute() error {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	return rootCmd.Execute()
}
