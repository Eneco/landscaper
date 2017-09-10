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
var env = &landscaper.Environment{}

var addCmd = &cobra.Command{
	Use:   "apply [files]...",
	Short: "Makes the current landscape match the desired landscape",
	RunE: func(cmd *cobra.Command, args []string) error {
		// setup env
		env.ComponentFiles = args

		if prefixDisable {
			env.ReleaseNamePrefix = ""
		} else {
			if env.ReleaseNamePrefix == "" {
				env.ReleaseNamePrefix = fmt.Sprintf("%s-", env.Namespace) // prefix not overridden; default to '<namespace>-'
			}
		}
		env.ChartLoader = landscaper.NewLocalCharts(env.HelmHome)

		v := landscaper.GetVersion()
		logrus.WithFields(logrus.Fields{"tag": v.GitTag, "commit": v.GitCommit}).Infof("This is Landscaper v%s", v.SemVer)
		logrus.WithFields(logrus.Fields{"namespace": env.Namespace, "releasePrefix": env.ReleaseNamePrefix, "dir": env.LandscapeDir, "dryRun": env.DryRun, "wait": env.Wait, "waitTimeout": env.WaitTimeout, "helmHome": env.HelmHome, "verbose": env.Verbose}).Info("Apply landscape desired state")

		// deprecated: populate ComponentFiles by getting *.yaml from LandscapeDir
		if len(args) == 0 && env.LandscapeDir != "" {
			logrus.Warnf("LandscapeDir is deprecated; please provide files as program arguments instead")
			env.ComponentFiles = []string{env.LandscapeDir}
		}

		kubeSecrets := landscaper.NewKubeSecretsReadWriteDeleter(env.KubeClient())
		envSecrets := landscaper.NewEnvironmentSecretsReader()
		fileState := landscaper.NewFileStateProvider(env.ComponentFiles, envSecrets, env.ChartLoader, env.ReleaseNamePrefix, env.Namespace)
		helmState := landscaper.NewHelmStateProvider(env.HelmClient(), kubeSecrets, env.ReleaseNamePrefix)
		executor := landscaper.NewExecutor(env.HelmClient(), env.ChartLoader, kubeSecrets, env.DryRun, env.Wait, int64(env.WaitTimeout / time.Second), env.DisabledStages)

		for {
			desired, err := fileState.Components()
			if err != nil {
				logrus.WithFields(logrus.Fields{"error": err}).Error("Loading desired state failed")
				return err
			}

			current, err := helmState.Components()
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

	landscapeNamespace := os.Getenv("LANDSCAPE_NAMESPACE")
	if landscapeNamespace == "" {
		landscapeNamespace = "default"
	}

	helmHome := os.ExpandEnv("$HOME/.helm")
	tillerNamespace := os.Getenv("TILLER_NAMESPACE")
	if tillerNamespace == "" {
		tillerNamespace = "kube-system"
	}

	f.BoolVar(&env.DryRun, "dry-run", false, "simulate the applying of the landscape. useful in merge requests")
	f.BoolVar(&env.Wait, "wait", false, "wait for all resources to be ready")
	f.DurationVar(&env.WaitTimeout, "wait-timeout", 5*time.Minute, "interval to wait for all resources to be ready")
	f.BoolVarP(&env.Verbose, "verbose", "v", false, "be verbose")
	f.BoolVar(&prefixDisable, "no-prefix", false, "disable prefixing release names")
	f.StringVar(&env.Context, "context", "", "the kube context to use. defaults to the current context")
	f.StringVar(&env.ReleaseNamePrefix, "prefix", landscapePrefix, "prefix release names with this string instead of <namespace>; overrides LANDSCAPE_PREFIX")
	f.StringVar(&env.LandscapeDir, "dir", landscapeDir, "(deprecated) path to a folder that contains all the landscape desired state files; overrides LANDSCAPE_DIR")
	f.StringVar(&env.Namespace, "namespace", landscapeNamespace, "namespace to apply the landscape to; overrides LANDSCAPE_NAMESPACE")
	f.StringVar(&env.HelmHome, "chart-dir", helmHome, "(deprecated; use --helm-home) Helm home directory")
	f.StringVar(&env.HelmHome, "helm-home", helmHome, "Helm home directory")
	f.StringVar(&env.TillerNamespace, "tiller-namespace", tillerNamespace, "Tiller namespace for Helm")
	f.Var(&env.DisabledStages, "disable", "Stages to be disabled")

	f.BoolVar(&env.Loop, "loop", false, "keep landscape in sync forever")
	f.DurationVar(&env.LoopInterval, "loop-interval", 5*time.Minute, "when running in a loop the interval between invocations")

	rootCmd.AddCommand(addCmd)
}
