package cmd

import (
	"io/ioutil"
	"log"

	"github.com/spf13/cobra"
)

var verbose bool = false

var RootCmd = &cobra.Command{
	Use:   "landscaper",
	Short: "A landscape desired state applicator",
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		if !verbose {
			log.SetOutput(ioutil.Discard)
		}
	},
}

func init() {
	RootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "be verbose")
}
