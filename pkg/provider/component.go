package provider

import (
	"errors"
	"fmt"
	"io/ioutil"

	"strings"

	"github.com/eneco/landscaper/pkg/landscaper"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
)

var (
	metadataKey       = "_metadata"
	releaseVersionKey = "releaseVersion"
	landscaperTagKey  = "landscaperControlled"

	// ErrNonLandscapeComponent is an error to indicate release is not controlled by landscaper
	ErrNonLandscapeComponent = errors.New("release is not controlled by landscaper")

	// ErrInvalidLandscapeMetadata is an error to indicate a release contains invalid landscaper metadata
	ErrInvalidLandscapeMetadata = errors.New("release contains invalid landscaper metadata")
)

type ComponentProvider interface {
	Current() ([]*landscaper.Component, error)
	Desired() ([]*landscaper.Component, error)
}

type componentProvider struct {
	env *landscaper.Environment
}

func (cp *componentProvider) Current() ([]*landscaper.Component, error) {
	return listComponentsFromFolder(cp.env.StateFolder, cp.env.RepositoryName)
}

func (cp *componentProvider) Desired() ([]*landscaper.Component, error) {
	return listComponentsFromCluster(cp.env)
}

func listComponentsFromFolder(path string, chartRepoName string) ([]*landscaper.Component, error) {
	components := []*landscaper.Component{}

	files, err := ioutil.ReadDir(path)
	if err != nil {
		return components, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		fmt.Println(file.Name())

		cmp, err := readComponentFromYAMLFilePath(file.Name())
		if err != nil {
			return components, err
		}

		err = coalesceComponent(cmp, chartRepoName)
		if err != nil {
			return components, err
		}

		components = append(components, cmp)
	}

	return components, nil
}

func listComponentsFromCluster(env *landscaper.Environment) ([]*landscaper.Component, error) {
	components := []*landscaper.Component{}

	// Retrieve the raw Helm release from the tiller
	helmReleases, err := listHelmReleases(env)
	if err != nil {
		return components, err
	}

	for _, release := range helmReleases {
		name := strings.TrimPrefix(release.Name, fmt.Sprintf("%s-", strings.ToLower(string(env.Name[0]))))

		cmp, err := newComponentFromHelmRelease(name, release)
		if err != nil {
			if err == ErrNonLandscapeComponent {
				continue
			}
			return components, err
		}

		components = append(components, cmp)
	}

	return components, nil
}

// newComponentFromYAML parses a byteslice into a Component instance
func newComponentFromYAML(content []byte) (*landscaper.Component, error) {
	cmp := &landscaper.Component{}
	if err := yaml.Unmarshal(content, cmp); err != nil {
		return nil, err
	}

	// Automatically add the component name as a configuration value as well
	cmp.Configuration["Name"] = cmp.Name

	return cmp, nil
}

func newComponentFromHelmRelease(name string, release *release.Release) (*landscaper.Component, error) {
	cfg, err := getReleaseConfiguration(release)
	if err != nil {
		return nil, err
	}

	metadata, err := getReleaseMetadata(cfg)
	if err != nil {
		return nil, err
	}

	delete(cfg, metadataKey)

	return &landscaper.Component{
		Name: name,
		Release: &landscaper.Release{
			Chart:   fmt.Sprintf("%s:%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version),
			Version: metadata[releaseVersionKey].(string),
		},
		Configuration: cfg,
	}, nil
}

// readComponentFromYAMLFilePath reads a yaml file from disk and returns a initialized Component
func readComponentFromYAMLFilePath(filePath string) (*landscaper.Component, error) {
	cfg, err := ioutil.ReadFile("../../test/component_test_data.yaml")
	if err != nil {
		return nil, err
	}

	return newComponentFromYAML(cfg)
}

// readComponentFromCluster reads the release, configuration and secrets from a k8s cluster for a specific component name
func readComponentFromCluster(name string, env *landscaper.Environment) (*landscaper.Component, error) {
	// Retrieve the raw Helm release from the tiller
	release, err := getHelmRelease(env.ReleaseName(name), env)
	if err != nil {
		return nil, err
	}

	return newComponentFromHelmRelease(name, release)
}

// coalesceComponent takes a component, loads the chart and coalesces the configuration with the default values
func coalesceComponent(cmp *landscaper.Component, chartRepoName string) error {
	ch, err := LoadChart(fmt.Sprintf("%s/%s", chartRepoName, cmp.Release.Chart))
	if err != nil {
		return err
	}

	raw, err := cmp.Configuration.YAML()
	if err != nil {
		return err
	}

	helmValues, err := chartutil.CoalesceValues(ch, &chart.Config{Raw: raw})
	if err != nil {
		return err
	}

	cmp.Configuration = landscaper.Configuration(helmValues)

	return nil
}

func listHelmReleases(env *landscaper.Environment) ([]*release.Release, error) {
	err := env.EnsureHelmClient()
	if err != nil {
		return nil, err
	}

	filter := helm.ReleaseListFilter(fmt.Sprintf("^%s-.+", strings.ToLower(string(env.Name[0]))))
	res, err := env.HelmClient.ListReleases(filter)
	if err != nil {
		return nil, err
	}

	return res.Releases, nil
}

func getHelmRelease(releaseName string, env *landscaper.Environment) (*release.Release, error) {
	err := env.EnsureHelmClient()
	if err != nil {
		return nil, err
	}

	res, err := env.HelmClient.ReleaseContent(releaseName)
	if err != nil {
		return nil, err
	}

	return res.Release, nil
}

func getReleaseConfiguration(helmRelease *release.Release) (landscaper.Configuration, error) {
	helmValues, err := chartutil.CoalesceValues(helmRelease.Chart, helmRelease.Config)
	if err != nil {
		return nil, err
	}

	return landscaper.Configuration(helmValues), nil
}

func getReleaseMetadata(cfg landscaper.Configuration) (map[string]interface{}, error) {
	val, ok := cfg[metadataKey]
	if !ok {
		return make(map[string]interface{}), ErrNonLandscapeComponent
	}

	metadata := val.(map[string]interface{})

	if _, ok := metadata[releaseVersionKey]; !ok {
		return nil, ErrInvalidLandscapeMetadata
	}
	if _, ok := metadata[landscaperTagKey]; !ok {
		return nil, ErrInvalidLandscapeMetadata
	}

	return metadata, nil
}
