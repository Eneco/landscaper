package landscaper

import (
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/services"
)

type HelmclientMock struct {
	deleteRelease  func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error)
	installRelease func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error)
	updateRelease  func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error)
}

func (_m *HelmclientMock) DeleteRelease(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
	return _m.deleteRelease(rlsName, opts...)
}

func (_m *HelmclientMock) GetVersion(opts ...helm.VersionOption) (*services.GetVersionResponse, error) {
	return nil, nil
}

func (_m *HelmclientMock) InstallRelease(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
	return _m.installRelease(chStr, namespace, opts...)
}

func (_m *HelmclientMock) ListReleases(opts ...helm.ReleaseListOption) (*services.ListReleasesResponse, error) {
	return nil, nil
}

func (_m *HelmclientMock) ReleaseContent(rlsName string, opts ...helm.ContentOption) (*services.GetReleaseContentResponse, error) {
	return nil, nil
}

func (_m *HelmclientMock) ReleaseHistory(rlsName string, opts ...helm.HistoryOption) (*services.GetHistoryResponse, error) {
	return nil, nil
}

func (_m *HelmclientMock) ReleaseStatus(rlsName string, opts ...helm.StatusOption) (*services.GetReleaseStatusResponse, error) {
	return nil, nil
}

func (_m *HelmclientMock) RollbackRelease(rlsName string, opts ...helm.RollbackOption) (*services.RollbackReleaseResponse, error) {
	return nil, nil
}

func (_m *HelmclientMock) UpdateRelease(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
	return _m.updateRelease(rlsName, chStr, opts...)
}

type MockChartLoader func(chartRef string) (*chart.Chart, string, error)

func (_m MockChartLoader) Load(chartRef string) (*chart.Chart, string, error) { return _m(chartRef) }
