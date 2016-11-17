package main

import (
	"fmt"

	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Write version information to stdout",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Printf("%#v\n", landscaper.GetVersion())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(versionCmd)
}
