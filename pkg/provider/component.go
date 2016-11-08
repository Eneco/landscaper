package provider

import (
	"fmt"
	"io/ioutil"

	"github.com/eneco/landscaper/pkg/landscaper"
	log "github.com/Sirupsen/logrus"
	"k8s.io/helm/pkg/chartutil"
)

// ReadComponentFromYAMLFilePath reads a yaml file from disk and returns a initialized Component
func ReadComponentFromYAMLFilePath(filePath string) (*landscaper.Component, error) {
	cfg, err := ioutil.ReadFile("../../test/component_test_data.yaml")
	if err != nil {
		return nil, err
	}

	return landscaper.NewComponentFromYAML(cfg)
}

// ReadComponentFromCluster reads the release, configuration and secrets from a k8s cluster for a specific component name
func ReadComponentFromCluster(componentName string, env *landscaper.Environment) (*landscaper.Component, error) {
	log.Error("Before ensure helm client")

	err := env.EnsureHelmClient()
	if err != nil {
		return nil, err
	}

	res, err := env.HelmClient.ReleaseContent(env.ReleaseName(componentName))
	if err != nil {
		return nil, err
	}

	cfg, err := chartutil.CoalesceValues(res.Release.Chart, res.Release.Config)
	if err != nil {
		return nil, err
	}

	fmt.Printf("%+v", cfg)

	return nil, nil
}
