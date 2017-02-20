package main

import (
	"fmt"
	"os"
	"time"

	"github.com/Sirupsen/logrus"
	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/spf13/cobra"
)

var prefixDisable bool

var addCmd = &cobra.Command{
	Use:   "apply",
	Short: "Makes the current landscape match the desired landscape",
	RunE: func(cmd *cobra.Command, args []string) error {
		// setup env
		if prefixDisable {
			env.ReleaseNamePrefix = ""
		} else {
			if env.ReleaseNamePrefix == "" {
				env.ReleaseNamePrefix = fmt.Sprintf("%s-", env.Namespace) // prefix not overridden; default to '<namespace>-'
			}
		}
		env.ChartLoader = landscaper.NewLocalCharts(env.ChartDir)

		v := landscaper.GetVersion()
		logrus.WithFields(logrus.Fields{"tag": v.GitTag, "commit": v.GitCommit}).Infof("This is Landscaper v%s", v.SemVer)
		logrus.WithFields(logrus.Fields{"namespace": env.Namespace, "releasePrefix": env.ReleaseNamePrefix, "dir": env.LandscapeDir, "dryRun": env.DryRun, "chartDir": env.ChartDir, "verbose": env.Verbose}).Info("Apply landscape desired state")

		sp := landscaper.NewSecretsProvider(env)
		cp := landscaper.NewComponentProvider(env, sp)
		executor := landscaper.NewExecutor(env, sp)

		for {
			desired, err := cp.Desired()
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Loading desired state failed")
				return err
			}

			current, err := cp.Current()
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Loading current state failed")
				return err
			}

			if err = executor.Apply(desired, current); err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Applying desired state failed")
				return err
			}

			if env.DryRun {
				logrus.Warn("Since dry-run is enabled, no actual actions have been performed")
			}

			if !env.Loop {
				break
			}

			logrus.Debugf("Running in a loop. Sleeping for %s.", env.LoopInterval)
			time.Sleep(env.LoopInterval)
		}

		return nil
	},
}

func init() {
	f := addCmd.Flags()

	landscapePrefix := os.Getenv("LANDSCAPE_PREFIX")

	landscapeDir := os.Getenv("LANDSCAPE_DIR")
	if landscapeDir == "" {
		landscapeDir = "."
	}

	landscapeNamespace := os.Getenv("LANDSCAPE_NAMESPACE")
	if landscapeNamespace == "" {
		landscapeNamespace = "default"
	}

	chartDir := os.ExpandEnv("$HOME/.helm")

	f.BoolVar(&env.DryRun, "dry-run", false, "simulate the applying of the landscape. useful in merge requests")
	f.BoolVarP(&env.Verbose, "verbose", "v", false, "be verbose")
	f.BoolVar(&prefixDisable, "no-prefix", false, "disable prefixing release names")
	f.StringVar(&env.Context, "context", "", "the kube context to use. defaults to the current context")
	f.StringVar(&env.ReleaseNamePrefix, "prefix", landscapePrefix, "prefix release names with this string instead of <namespace>; overrides LANDSCAPE_PREFIX")
	f.StringVar(&env.LandscapeDir, "dir", landscapeDir, "path to a folder that contains all the landscape desired state files; overrides LANDSCAPE_DIR")
	f.StringVar(&env.Namespace, "namespace", landscapeNamespace, "namespace to apply the landscape to; overrides LANDSCAPE_NAMESPACE")
	f.StringVar(&env.ChartDir, "chart-dir", chartDir, "where the charts are stored")
	f.BoolVar(&env.NoCronUpdate, "no-cronjob-update", false, "replaces CronJob updates with a create+delete; k8s #35149 work around")
	f.BoolVar(&env.Loop, "loop", false, "keep landscape in sync forever")
	f.DurationVar(&env.LoopInterval, "loop-interval", 5*time.Minute, "when running in a loop the interval between invocations")

	rootCmd.AddCommand(addCmd)
}
