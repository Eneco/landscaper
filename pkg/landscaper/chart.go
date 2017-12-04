package landscaper

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/sirupsen/logrus"

	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/downloader"
	"k8s.io/helm/pkg/getter"
	"k8s.io/helm/pkg/helm/environment"
	"k8s.io/helm/pkg/helm/helmpath"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

// ChartLoader allows one to load Charts by name
type ChartLoader interface {
	Load(chartRef string) (*chart.Chart, string, error)
}

// LocalCharts allows one to load Charts from a local path
type LocalCharts struct {
	HomePath string
}

// NewLocalCharts creates a LocalCharts ChartLoader
func NewLocalCharts(homePath string) *LocalCharts {
	return &LocalCharts{HomePath: homePath}
}

// Load locates, and potentially downloads, a chart to the local repository
func (c *LocalCharts) Load(chartRef string) (*chart.Chart, string, error) {
	logrus.WithFields(logrus.Fields{"chartRef": chartRef}).Debug("Load Chart")

	chartPath, err := locateChartPath(c.HomePath, chartRef)
	if err != nil {
		return nil, "", err
	}

	chart, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, "", err
	}

	logrus.WithFields(logrus.Fields{"chartRef": chartRef}).Debug("Loaded Chart successfully")
	return chart, chartPath, nil
}

// locateChartPath downloads charts by reference. It stores the resulting tgzs in a temporary directory.
func locateChartPath(homePath, chartRef string) (string, error) {
	name, version := parseChartRef(chartRef)
	logrus.WithFields(logrus.Fields{"chartRef": chartRef, "homePath": homePath, "name": name, "version": version}).Debug("locateChartPath")

	repoName := ""
	info := strings.Split(name, "/")
	if len(info) != 2 {
		return "", fmt.Errorf("expect repo/name instead of `%s`", name)
	}
	repoName = info[0]

	dl := downloader.ChartDownloader{
		HelmHome: helmpath.Home(homePath),
		Out:      os.Stdout,
		Getters:  getter.All(environment.EnvSettings{}),
	}

	// ResolveChartVersion provides us through the repo index an url from which we can obtain the filename chart.tgz
	url, _, err := dl.ResolveChartVersion(name, version)
	if err != nil {
		return "", fmt.Errorf("cannot resolve chartversion: %s", err)
	}

	_, chartFile := filepath.Split(url.Path)

	repoDir := filepath.Join(os.TempDir(), "landscaper", repoName)
	if err := os.MkdirAll(repoDir, 0755); err != nil {
		return "", fmt.Errorf("cannot create work directory `%s`", repoDir)
	}

	chartPath, err := filepath.Abs(filepath.Join(repoDir, chartFile))
	if err != nil {
		return "", err
	}

	logrus.WithFields(logrus.Fields{"chartPath": chartPath}).Debug("Look for cached local package")

	if _, err := os.Stat(chartPath); err == nil {
		return chartPath, nil
	}

	logrus.WithFields(logrus.Fields{"name": name, "version": version, "repoDir": repoDir}).Debug("Download")
	_, _, err = dl.DownloadTo(name, version, repoDir)
	if err != nil {
		return "", fmt.Errorf("failed to download `%s`: %s", chartRef, err)
	}

	return chartPath, nil
}

// parseChartRef splits a name:version into a name and an (optional) version
func parseChartRef(ref string) (string, string) {
	chartInfo := strings.Split(ref, ":")
	chartName, chartVersion := chartInfo[0], ""
	if len(chartInfo) == 2 {
		chartVersion = chartInfo[1]
	}

	return strings.TrimSpace(chartName), strings.TrimSpace(chartVersion)
}
