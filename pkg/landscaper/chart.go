package landscaper

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/cmd/helm/downloader"
	"k8s.io/helm/cmd/helm/helmpath"
	"k8s.io/helm/pkg/chartutil"
	"k8s.io/helm/pkg/proto/hapi/chart"
)

type ChartLoader interface {
	Load(chartRef string) (*chart.Chart, string, error)
}

type LocalCharts struct {
	HomePath string
}

func NewLocalCharts(homePath string) *LocalCharts {
	return &LocalCharts{HomePath: homePath}
}

// ErrChartNotFound is thrown when an unknown chart is trying to be loaded
var ErrChartNotFound = errors.New("chart not found")

// Load locates, and potentially downloads, a chart to the local repository
func (c *LocalCharts) Load(chartRef string) (*chart.Chart, string, error) {
	chartPath, err := locateChartPath(c.HomePath, chartRef)
	if err != nil {
		return nil, "", err
	}

	chart, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, "", err
	}

	return chart, chartPath, nil
}

func locateChartPath(homePath, chartRef string) (string, error) {
	name, version := parseChartRef(chartRef)

	chartFile := filepath.Join(helmpath.Home(homePath).Repository(), name)
	if _, err := os.Stat(chartFile); err == nil {
		return filepath.Abs(chartFile)
	}

	dl := downloader.ChartDownloader{
		HelmHome: helmpath.Home(homePath),
		Out:      os.Stdout,
	}

	chartFile, _, err := dl.DownloadTo(name, version, helmpath.Home(homePath).Repository())
	if err == nil {
		chartFile, err = filepath.Abs(chartFile)
		if err != nil {
			return "", err
		}

		repoName := ""
		info := strings.Split(name, "/")
		if len(info) == 2 {
			repoName = info[0]
		}

		// Extract the chart for easier reference the next time
		chartutil.ExpandFile(filepath.Join(helmpath.Home(homePath).Repository(), repoName), chartFile)

		return chartFile, nil
	}

	return "", ErrChartNotFound
}

func parseChartRef(ref string) (string, string) {
	chartInfo := strings.Split(ref, ":")
	chartName, chartVersion := chartInfo[0], ""
	if len(chartInfo) == 2 {
		chartVersion = chartInfo[1]
	}

	return strings.TrimSpace(chartName), strings.TrimSpace(chartVersion)
}
