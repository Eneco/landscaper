package cmd

import (
	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/eneco/landscaper/pkg/provider"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "apply <arguments>", //TODO: define exact arguments
	Short: "Makes the current landscape match the desired landscape",
	RunE: func(cmd *cobra.Command, args []string) error {
		_, err := provider.ReadComponentFromCluster("traefik", &landscaper.Environment{
			Name:      "test",
			Namespace: "landscaper-testing",
		})
		if err != nil {
			return err
		}

		return nil
	},
}

func init() {
	RootCmd.AddCommand(addCmd)
}
