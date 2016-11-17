package main

import (
	"os"
	"time"

	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
)

var env = &landscaper.Environment{}

var (
	dryRun = false
)

var rootCmd = &cobra.Command{
	Use:   "landscaper",
	Short: "A landscape desired state applicator",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		if env.Verbose {
			logrus.SetLevel(logrus.DebugLevel)
		}
		return env.EnsureHelmClient()
	},
	SilenceUsage: true,
	// @TODO: figure out if the following is needed?!
	// PersistentPostRun: func(cmd *cobra.Command, args []string) {
	// 	env.Teardown()
	// },
}

func init() {
	_ = rootCmd.PersistentFlags()
	p := &prefixed.TextFormatter{}
	p.TimestampFormat = time.RFC3339
	logrus.SetFormatter(p)
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}
