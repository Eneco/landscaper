package landscaper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"path/filepath"

	"github.com/Sirupsen/logrus"
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
	env             *Environment
	secretsProvider SecretsProvider
}

// NewComponentProvider is a factory method to create a new ComponentProvider
func NewComponentProvider(env *Environment, secretsProvider SecretsProvider) ComponentProvider {
	return &componentProvider{
		env:             env,
		secretsProvider: secretsProvider,
	}
}

// Current returns all Components in the cluster
func (cp *componentProvider) Current() ([]*Component, error) {
	components := []*Component{}

	logrus.Info("Obtain current state Helm Releases (Components) from Tiller")

	// Retrieve the raw Helm release from the tiller
	helmReleases, err := cp.listHelmReleases()
	if err != nil {
		return components, err
	}

	for _, release := range helmReleases {
		cmp, err := newComponentFromHelmRelease(release)
		if err == ErrNonLandscapeComponent {
			continue
		}
		if err != nil {
			return components, err
		}

		secretValues, err := cp.secretsProvider.Read(cmp.Name)
		if err != nil {
			return components, err
		}

		cmp.Secrets = Secrets{}
		for key := range secretValues {
			cmp.Secrets = append(cmp.Secrets, key)
		}

		components = append(components, cmp)
	}

	logrus.WithFields(logrus.Fields{"totalReleases": len(helmReleases), "landscapedComponents": len(components)}).Info("Retrieved Releases (Components)")

	return components, nil
}

// Desired returns all desired components according to their descriptions
func (cp *componentProvider) Desired() ([]*Component, error) {
	components := []*Component{}

	logrus.WithFields(logrus.Fields{"directory": cp.env.LandscapeDir}).Info("Obtain desired state from directory")

	files, err := ioutil.ReadDir(cp.env.LandscapeDir)
	if err != nil {
		return components, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		filename := filepath.Join(cp.env.LandscapeDir, file.Name())

		logrus.WithFields(logrus.Fields{"directory": cp.env.LandscapeDir, "file": file.Name()}).Debug("Read desired state from file")
		cmp, err := readComponentFromYAMLFilePath(filename)
		if err != nil {
			return components, err
		}

		err = cp.coalesceComponent(cmp)
		if err != nil {
			return components, err
		}

		cmp.Configuration["Name"] = cmp.Name
		cmp.Name = cp.env.ReleaseName(cmp.Name)

		readSecretValues(cmp)

		if err := cmp.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate `%s`: %s", filename, err)
		}

		components = append(components, cmp)
	}

	if err := validateComponents(components); err != nil {
		return nil, err
	}

	logrus.WithFields(logrus.Fields{"directory": cp.env.LandscapeDir, "components": len(components)}).Debug("Desired state has been read")

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

// coalesceComponent takes a component, loads the chart and coalesces the configuration with the default values
func (cp *componentProvider) coalesceComponent(cmp *Component) error {
	logrus.WithFields(logrus.Fields{"chart": cmp.Release.Chart}).Debug("coalesceComponent")
	ch, _, err := cp.env.ChartLoader.Load(fmt.Sprintf("%s/%s", cp.env.HelmRepositoryName, cmp.Release.Chart))
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

// listHelmReleases lists all releases that are prefixed with env.LandscapeName
func (cp *componentProvider) listHelmReleases() ([]*release.Release, error) {
	logrus.Debug("listHelmReleases")
	filter := helm.ReleaseListFilter(fmt.Sprintf("^%s.+", cp.env.ReleaseNamePrefix()))
	res, err := cp.env.HelmClient().ListReleases(filter)
	if err != nil {
		return nil, err
	}

	return res.Releases, nil
}

// getHelmRelease gets a Release
func (cp *componentProvider) getHelmRelease(releaseName string) (*release.Release, error) {
	logrus.WithFields(logrus.Fields{"releaseName": releaseName}).Debug("getHelmRelease")
	res, err := cp.env.HelmClient().ReleaseContent(releaseName)
	if err != nil {
		return nil, err
	}

	return res.Release, nil
}

// newComponentFromHelmRelease creates a Component from a Release
func newComponentFromHelmRelease(release *release.Release) (*Component, error) {
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
		release.Name,
		&Release{
			Chart:   fmt.Sprintf("%s:%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version),
			Version: metadata[releaseVersionKey].(string),
		},
		cfg,
		Secrets{},
	), nil
}

// readComponentFromYAMLFilePath reads a yaml file from disk and returns an initialized Component
func readComponentFromYAMLFilePath(filePath string) (*Component, error) {
	cfg, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return newComponentFromYAML(cfg)
}

// getReleaseConfiguration returns a release's coalesced Cnfiguration (= helm values)
func getReleaseConfiguration(helmRelease *release.Release) (Configuration, error) {
	helmValues, err := chartutil.CoalesceValues(helmRelease.Chart, helmRelease.Config)
	if err != nil {
		return nil, err
	}

	return Configuration(helmValues), nil
}

// getReleaseMetadata extracts landscaper's metadata from a Configuration
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
