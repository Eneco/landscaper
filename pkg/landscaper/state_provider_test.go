package landscaper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
)

func TestHelmStateProviderComponents(t *testing.T) {
	helmMock := &HelmclientMock{
		listReleases: func(opts ...helm.ReleaseListOption) (*services.ListReleasesResponse, error) {
			t.Logf("listReleases %#v", opts)
			rels := &services.ListReleasesResponse{
				Releases: []*release.Release{
					&release.Release{
						Name:      "my-release",
						Namespace: "my-namespace",
						Chart: &chart.Chart{
							Metadata: &chart.Metadata{
								Name:    "chart-name",
								Version: "1.3.37",
							},
							Values: &chart.Config{Raw: `
config_a: xxx
config_b: yyy
`},
						},
						Config: &chart.Config{Raw: fmt.Sprintf(
							`%s:
  %s: 1.2.3
  %s: repo1
config_b: zzz
config_c: qqq
`, metadataKey, metaReleaseVersion, metaChartRepo)},
					},
				},
			}
			return rels, nil
		},
	}
	secretsMock := SecretsProviderMock{
		read: func(componentName, namespace string, secretNames []string) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			return nil, nil
		},
	}

	hs := NewHelmStateProvider(helmMock, secretsMock, "my-prefix")
	cmps, err := hs.Components()
	require.NoError(t, err)
	require.Contains(t, cmps, "my-release")
	c := cmps["my-release"]
	require.Equal(t, c.Name, "my-release")
	require.Equal(t, c.Namespace, "my-namespace")
	require.Equal(t, c.Configuration["config_a"], "xxx") // from chart
	require.Equal(t, c.Configuration["config_b"], "zzz") // from chart but overridden in values
	require.Equal(t, c.Configuration["config_c"], "qqq") // in values but not in chart
}
