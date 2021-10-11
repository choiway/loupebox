package cmd

import (
	"github.com/spf13/cobra"
)

var (
	// Used for flags.
	cfgFile     string
	userLicense string

	rootCmd = &cobra.Command{
		Use: "loupebox",
		Short: "Photo management utility",
		Long: `Loupebox is a cli application that helps you aggregate your photos 
in an organized manner. It helps you avoid copying duplicates and organizes
your photos by the date they were taken.
`,
	}
)

// Execute executes the root command.
func Execute() error {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	return rootCmd.Execute()
}
