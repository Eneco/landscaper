package main

import (
	"os"

	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/Sirupsen/logrus"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "apply",
	Short: "Makes the current landscape match the desired landscape",
	RunE: func(cmd *cobra.Command, args []string) error {
		logrus.WithFields(logrus.Fields{"version": landscaper.GetVersion(), "namespace": env.Namespace, "landscapeName": env.LandscapeName, "repo": env.HelmRepositoryName, "dir": env.LandscapeDir, "dryRun": env.DryRun}).Info("Apply landscape desired state")

		sp := landscaper.NewSecretsProvider(env)
		cp := landscaper.NewComponentProvider(env, sp)
		executor := landscaper.NewExecutor(env, sp)

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

		return nil
	},
}

func init() {
	f := addCmd.Flags()

	helmRepositoryName := os.Getenv("HELM_REPOSITORY_NAME")
	if helmRepositoryName == "" {
		helmRepositoryName = "eet"
	}

	landscapeName := os.Getenv("LANDSCAPE_NAME")
	if landscapeName == "" {
		landscapeName = "acceptance"
	}

	landscapeDir := os.Getenv("LANDSCAPE_DIR")
	if landscapeDir == "" {
		landscapeDir = "."
	}

	landscapeNamespace := os.Getenv("LANDSCAPE_NAMESPACE")
	if landscapeNamespace == "" {
		landscapeNamespace = "acceptance"
	}

	env.ChartLoader = landscaper.NewLocalCharts(os.ExpandEnv("$HOME/.helm"))

	f.BoolVar(&env.DryRun, "dry-run", false, "simulate the applying of the landscape. useful in merge requests")
	f.BoolVarP(&env.Verbose, "verbose", "v", false, "be verbose")
	f.StringVar(&env.HelmRepositoryName, "helm-repo-name", helmRepositoryName, "the name of the helm repository that contains all the charts")
	f.StringVar(&env.LandscapeName, "landscape-name", landscapeName, "name of the landscape. the first letter of this is used as a prefix, e.g. acceptance creates releases like a-release-name")
	f.StringVar(&env.LandscapeDir, "landscape-dir", landscapeDir, "path to a folder that contains all the landscape desired state files")
	f.StringVar(&env.Namespace, "landscape-namespace", landscapeNamespace, "namespace to apply the landscape to")

	rootCmd.AddCommand(addCmd)
}
