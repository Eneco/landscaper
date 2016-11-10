package main

import (
	"os"

	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/spf13/cobra"
)

var env = &landscaper.Environment{}

var (
	dryRun = false
)

var rootCmd = &cobra.Command{
	Use:   "landscaper",
	Short: "A landscape desired state applicator",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		_ = env.EnsureHelmClient()
	},
	// @TODO: figure out if the following is needed?!
	// PersistentPostRun: func(cmd *cobra.Command, args []string) {
	// 	env.Teardown()
	// },
}

func init() {
	_ = rootCmd.PersistentFlags()
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
