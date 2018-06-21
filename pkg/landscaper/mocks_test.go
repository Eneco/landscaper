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
	listReleases   func(opts ...helm.ReleaseListOption) (*services.ListReleasesResponse, error)
}

func (m *HelmclientMock) ListReleases(opts ...helm.ReleaseListOption) (*services.ListReleasesResponse, error) {
	return m.listReleases(opts...)
}

func (m *HelmclientMock) InstallRelease(chStr, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
	return m.installRelease(chStr, namespace, opts...)
}

func (m *HelmclientMock) InstallReleaseFromChart(chart *chart.Chart, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) DeleteRelease(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
	return m.deleteRelease(rlsName, opts...)
}

func (m *HelmclientMock) ReleaseStatus(rlsName string, opts ...helm.StatusOption) (*services.GetReleaseStatusResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) UpdateRelease(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
	return m.updateRelease(rlsName, chStr, opts...)
}

func (m *HelmclientMock) UpdateReleaseFromChart(rlsName string, chart *chart.Chart, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) RollbackRelease(rlsName string, opts ...helm.RollbackOption) (*services.RollbackReleaseResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) ReleaseContent(rlsName string, opts ...helm.ContentOption) (*services.GetReleaseContentResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) ReleaseHistory(rlsName string, opts ...helm.HistoryOption) (*services.GetHistoryResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) GetVersion(opts ...helm.VersionOption) (*services.GetVersionResponse, error) {
	return nil, nil
}

func (m *HelmclientMock) RunReleaseTest(rlsName string, opts ...helm.ReleaseTestOption) (<-chan *services.TestReleaseResponse, <-chan error) {
	return nil, nil
}

func (m *HelmclientMock) PingTiller() error {
	return nil
}

type MockChartLoader func(chartRef string) (*chart.Chart, string, error)

func (m MockChartLoader) Load(chartRef string) (*chart.Chart, string, error) { return m(chartRef) }

type SecretsProviderMock struct {
	write  func(releaseName, namespace string, values SecretValues) error
	read   func(releaseName, namespace string, secretNames SecretNames) (SecretValues, error)
	delete func(releaseName, namespace string) error
}

func (m SecretsProviderMock) Write(releaseName, namespace string, values SecretValues) error {
	return m.write(releaseName, namespace, values)
}

func (m SecretsProviderMock) Read(releaseName, namespace string, secretNames SecretNames) (SecretValues, error) {
	return m.read(releaseName, namespace, secretNames)
}

func (m SecretsProviderMock) Delete(releaseName, namespace string) error {
	return m.delete(releaseName, namespace)
}
