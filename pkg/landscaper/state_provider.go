package landscaper

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Sirupsen/logrus"
	validator "gopkg.in/validator.v2"
	"gopkg.in/yaml.v2"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
)

var (
	// ErrNonLandscapeComponent is an error to indicate a release is not controlled by landscaper
	ErrNonLandscapeComponent = errors.New("release is not controlled by landscaper")

	// ErrInvalidLandscapeMetadata is an error to indicate a release contains invalid landscaper metadata
	ErrInvalidLandscapeMetadata = errors.New("release contains invalid landscaper metadata")
)

// StateProvider can be used to obtain a state, actual (from Helm) or desired (e.g. from files)
type StateProvider interface {
	Components() (Components, error)
}

type fileStateProvider struct {
	fileNames         []string
	secrets           SecretsReader
	chartLoader       ChartLoader
	releaseNamePrefix string
	namespace         string
}

type helmStateProvider struct {
	helmClient        helm.Interface
	secrets           SecretsReader
	releaseNamePrefix string
}

// NewFileStateProvider creates a StateProvider that sources Files
func NewFileStateProvider(fileNames []string, secrets SecretsReader, chartLoader ChartLoader, releaseNamePrefix, namespace string) StateProvider {
	return &fileStateProvider{fileNames, secrets, chartLoader, releaseNamePrefix, namespace}
}

// NewHelmStateProvider creates a StateProvider that sources Helm (actual state)
func NewHelmStateProvider(helmClient helm.Interface, secrets SecretsReader, releaseNamePrefix string) StateProvider {
	return &helmStateProvider{helmClient, secrets, releaseNamePrefix}
}

// Components returns all Components in the cluster
func (cp *helmStateProvider) Components() (Components, error) {
	components := Components{}

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

		secretValues, err := cp.secrets.Read(cmp.Name, release.Namespace, nil)
		if err != nil {
			return components, err
		}

		cmp.SecretValues = secretValues
		cmp.Secrets = Secrets{}

		for key := range secretValues {
			cmp.Secrets = append(cmp.Secrets, key)
		}
		sort.Strings(cmp.Secrets) // enforce a consistent ordering for proper diffing / deepEqualing

		components[cmp.Name] = cmp
	}

	logrus.WithFields(logrus.Fields{"totalReleases": len(helmReleases), "landscapedComponents": len(components)}).Info("Retrieved Releases (Components)")

	return components, nil
}

// get loads the provided files. If the argument is a directory, *.yaml in it is loaded.
func (cp *fileStateProvider) get(files []string) (Components, error) {
	components := Components{}

	logrus.WithFields(logrus.Fields{"files": files}).Info("Obtain desired state from files")

	for _, filename := range files {
		fileInfo, err := os.Stat(filename)
		if err != nil {
			return nil, err
		}
		if fileInfo.IsDir() {
			logrus.WithFields(logrus.Fields{"file": filename}).Debugf("Crawl directory for *.yaml")
			files, err := filepath.Glob(filepath.Join(filename, "*.yaml"))
			if err != nil {
				return nil, err
			}
			subComp, err := cp.get(files)
			if err != nil {
				return nil, err
			}
			for k, v := range subComp { // TODO: check for duplicate names
				components[k] = v
			}
			continue
		}

		logrus.WithFields(logrus.Fields{"file": filename}).Debug("Read desired state from file")
		cmp, err := readComponentFromYAMLFilePath(filename)
		if err != nil {
			return nil, fmt.Errorf("readComponentFromYAMLFilePath file `%s` failed: %s", filename, err)
		}
		cp.normalizeFromFile(cmp)

		err = cp.coalesceComponent(cmp)
		if err != nil {
			return nil, err
		}

		if len(cmp.Secrets) > 0 {
			sort.Strings(cmp.Secrets) // enforce a consistent ordering for proper diffing / deepEqualing
			secr, err := cp.secrets.Read(cmp.Name, cmp.Namespace, []string(cmp.Secrets))
			if err != nil {
				return nil, err
			}
			cmp.SecretValues = secr
		}

		if err := cmp.Validate(); err != nil {
			return nil, fmt.Errorf("failed to validate `%s`: %s", filename, err)
		}

		// make sure there are no duplicate names
		if _, ok := components[cmp.Name]; ok {
			return nil, fmt.Errorf("duplicate component name `%s`", cmp.Name)
		}

		logrus.Debugf("desired %#v", *cmp)

		components[cmp.Name] = cmp
	}

	if err := validateComponents(components); err != nil {
		return components, err
	}

	logrus.WithFields(logrus.Fields{"n_components": len(components)}).Debug("Desired state has been read")

	return components, nil
}

// normalizeFromFile makes a Component look identical to a Component reconstructed from Helm
func (cp *fileStateProvider) normalizeFromFile(c *Component) error {
	c.Configuration["Name"] = c.Name
	c.Name = cp.releaseNamePrefix + strings.ToLower(c.Name)
	if len(c.Secrets) > 0 {
		c.Configuration["secretsRef"] = c.Name
	}

	ss := strings.Split(c.Release.Chart, "/")
	if len(ss) != 2 {
		return fmt.Errorf("bad release.chart: `%s`, expecting `some_repo/some_name`", c.Release.Chart)
	}
	c.Release.Chart = ss[1]

	c.Configuration.SetMetadata(&Metadata{ChartRepository: ss[0], ReleaseVersion: c.Release.Version})

	if c.Namespace == "" {
		c.Namespace = cp.namespace
	}

	// when the chart ref is versioned, we're done
	if strings.Contains(c.Release.Chart, ":") {
		return nil
	}

	// when the chart ref is not versioned, set it to the latest chart
	chartRef, err := c.FullChartRef()
	if err != nil {
		return err
	}
	ch, _, err := cp.chartLoader.Load(chartRef)
	if err != nil {
		return err
	}
	c.Release.Chart = fmt.Sprintf("%s:%s", c.Release.Chart, ch.Metadata.Version)
	return nil
}

// Get returns all desired components according to their descriptions
func (cp *fileStateProvider) Components() (Components, error) {
	return cp.get(cp.fileNames)
}

// newComponentFromYAML parses a byteslice into a Component instance
func newComponentFromYAML(content []byte) (*Component, error) {
	cmp := &Component{}
	if err := yaml.Unmarshal(content, cmp); err != nil {
		return nil, err
	}

	if cmp.Name == "" {
		return nil, errors.New("invalid input yaml; name missing")
	}

	if cmp.Release == nil {
		return nil, errors.New("invalid input yaml; release missing")
	}

	if err := validator.Validate(cmp.Release); err != nil {
		return nil, err
	}

	return NewComponent(cmp.Name, cmp.Namespace, cmp.Release, cmp.Configuration, cmp.Secrets), nil
}

// coalesceComponent takes a component, loads the chart and coalesces the configuration with the default values
func (cp *fileStateProvider) coalesceComponent(cmp *Component) error {
	logrus.WithFields(logrus.Fields{"chart": cmp.Release.Chart}).Debug("coalesceComponent")
	chartRef, err := cmp.FullChartRef()
	if err != nil {
		return err
	}
	ch, _, err := cp.chartLoader.Load(chartRef)
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

// listHelmReleases lists all releases that are prefixed with releaseNamePrefix
func (cp *helmStateProvider) listHelmReleases() ([]*release.Release, error) {
	logrus.Debug("listHelmReleases")
	filter := helm.ReleaseListFilter(fmt.Sprintf("^%s.+", cp.releaseNamePrefix))
	res, err := cp.helmClient.ListReleases(filter)
	if err != nil {
		return nil, err
	}

	return res.Releases, nil
}

// newComponentFromHelmRelease creates a Component from a Release
func newComponentFromHelmRelease(release *release.Release) (*Component, error) {
	cfg, err := getReleaseConfiguration(release)
	if err != nil {
		return nil, err
	}

	if !cfg.HasMetadata() {
		return nil, ErrNonLandscapeComponent
	}

	m, err := cfg.GetMetadata()
	if err != nil {
		return nil, err
	}

	cmp := NewComponent(
		release.Name,
		release.Namespace,
		&Release{
			Chart:   fmt.Sprintf("%s:%s", release.Chart.Metadata.Name, release.Chart.Metadata.Version),
			Version: m.ReleaseVersion,
		},
		cfg,
		Secrets{},
	)

	return cmp, nil
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
