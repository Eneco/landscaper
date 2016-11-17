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
		logrus.WithFields(logrus.Fields{"namespace": env.Namespace, "landscapeName": env.LandscapeName, "repo": env.HelmRepositoryName, "dir": env.LandscapeDir, "dryRun": env.DryRun}).Info("Apply landscape desired state")

		cp, err := landscaper.NewComponentProvider(env)
		if err != nil {
			return err
		}

		exec, err := landscaper.NewExecutor(env)
		if err != nil {
			return err
		}

		desired, err := cp.Desired()
		if err != nil {
			return err
		}

		current, err := cp.Current()
		if err != nil {
			return err
		}

		err = exec.Apply(desired, current)
		if err != nil {
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
	f.StringVar(&env.HelmRepositoryName, "helm-repo-name", helmRepositoryName, "the name of the helm repository that contains all the charts")
	f.StringVar(&env.LandscapeName, "landscape-name", landscapeName, "name of the landscape. the first letter of this is used as a prefix, e.g. acceptance creates releases like a-release-name")
	f.StringVar(&env.LandscapeDir, "landscape-dir", landscapeDir, "path to a folder that contains all the landscape desired state files")
	f.StringVar(&env.Namespace, "landscape-namespace", landscapeNamespace, "namespace to apply the landscape to")

	rootCmd.AddCommand(addCmd)
}
