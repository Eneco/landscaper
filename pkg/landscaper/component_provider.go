package landscaper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"

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

// ComponentProvider can be used to interact with components locally, as well as on the cluster
type ComponentProvider interface {
	Current() ([]*Component, error)
	Desired() ([]*Component, error)
}

type componentProvider struct {
	env *Environment
}

// NewComponentProvider is a factory method to create a new ComponentProvider
func NewComponentProvider(env *Environment) (ComponentProvider, error) {
	err := env.EnsureHelmClient()
	if err != nil {
		return nil, err
	}

	return &componentProvider{env: env}, nil
}

func (cp *componentProvider) Current() ([]*Component, error) {
	components := []*Component{}

	// Retrieve the raw Helm release from the tiller
	helmReleases, err := cp.listHelmReleases()
	if err != nil {
		return components, err
	}

	for _, release := range helmReleases {
		name := strings.TrimPrefix(release.Name, fmt.Sprintf("%s-", strings.ToLower(string(cp.env.LandscapeName[0]))))

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

func (cp *componentProvider) Desired() ([]*Component, error) {
	components := []*Component{}

	files, err := ioutil.ReadDir(cp.env.LandscapeDir)
	if err != nil {
		return components, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		cmp, err := readComponentFromYAMLFilePath(filepath.Join(cp.env.LandscapeDir, file.Name()))
		if err != nil {
			return components, err
		}

		err = cp.coalesceComponent(cmp)
		if err != nil {
			return components, err
		}

		components = append(components, cmp)
	}

	return components, nil
}

// newComponentFromYAML parses a byteslice into a Component instance
func newComponentFromYAML(content []byte) (*Component, error) {
	cmp := &Component{}
	if err := yaml.Unmarshal(content, cmp); err != nil {
		return nil, err
	}

	return NewComponent(cmp.Name, cmp.Release, cmp.Configuration, cmp.Secrets), nil
}

// readComponentFromCluster reads the release, configuration and secrets from a k8s cluster for a specific component name
func (cp *componentProvider) readComponentFromCluster(name string, env *Environment) (*Component, error) {
	// Retrieve the raw Helm release from the tiller
	release, err := cp.getHelmRelease(env.ReleaseName(name))
	if err != nil {
		return nil, err
	}

	return newComponentFromHelmRelease(name, release)
}

// coalesceComponent takes a component, loads the chart and coalesces the configuration with the default values
func (cp *componentProvider) coalesceComponent(cmp *Component) error {
	ch, _, err := LoadChart(fmt.Sprintf("%s/%s", cp.env.HelmRepositoryName, cmp.Release.Chart))
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

	cmp.Configuration = Configuration(helmValues)

	return nil
}

func (cp *componentProvider) listHelmReleases() ([]*release.Release, error) {
	filter := helm.ReleaseListFilter(fmt.Sprintf("^%s-.+", strings.ToLower(string(cp.env.LandscapeName[0]))))
	res, err := cp.env.HelmClient.ListReleases(filter)
	if err != nil {
		return nil, err
	}

	return res.Releases, nil
}

func (cp *componentProvider) getHelmRelease(releaseName string) (*release.Release, error) {
	res, err := cp.env.HelmClient.ReleaseContent(releaseName)
	if err != nil {
		return nil, err
	}

	return res.Release, nil
}

func newComponentFromHelmRelease(name string, release *release.Release) (*Component, error) {
	cfg, err := getReleaseConfiguration(release)
	if err != nil {
		return nil, err
	}

	metadata, err := getReleaseMetadata(cfg)
	if err != nil {
		return nil, err
	}

	delete(cfg, metadataKey)

	return NewComponent(
		name,
		&Release{
			Chart:   fmt.Sprintf("%s:%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version),
			Version: metadata[releaseVersionKey].(string),
		},
		cfg,
		nil,
	), nil
}

// readComponentFromYAMLFilePath reads a yaml file from disk and returns a initialized Component
func readComponentFromYAMLFilePath(filePath string) (*Component, error) {
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return newComponentFromYAML(cfg)
}

func getReleaseConfiguration(helmRelease *release.Release) (Configuration, error) {
	helmValues, err := chartutil.CoalesceValues(helmRelease.Chart, helmRelease.Config)
	if err != nil {
		return nil, err
	}

	return Configuration(helmValues), nil
}

func getReleaseMetadata(cfg Configuration) (map[string]interface{}, error) {
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
