package provider

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

var ErrChartNotFound = errors.New("chart not found")

func LoadChart(chartRef string) (*chart.Chart, error) {
	chartPath, err := locateChartPath(chartRef)
	if err != nil {
		return nil, err
	}

	chart, err := chartutil.Load(chartPath)
	if err != nil {
		return nil, err
	}

	return chart, nil
}

func locateChartPath(chartRef string) (string, error) {
	name, version := parseChartRef(chartRef)
	homePath := os.ExpandEnv("$HOME/.helm")

	if _, err := os.Stat(name); err == nil {
		abs, err := filepath.Abs(name)
		if err != nil {
			return abs, err
		}
		return abs, nil
	}

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
