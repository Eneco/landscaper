package landscaper

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/release"
	"k8s.io/helm/pkg/proto/hapi/services"
)

func TestFileStateProviderComponents(t *testing.T) {
	secretsMock := SecretsProviderMock{
		read: func(componentName, namespace string, secretNames []string) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			vs := SecretValues{}
			for _, s := range secretNames {
				vs[s] = []byte(componentName + namespace + strings.Replace(s, "e", "3", -1))
			}
			return vs, nil
		},
	}

	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		c := &chart.Chart{
			Metadata: &chart.Metadata{
				Name:    "chart-name",
				Version: "1.3.37",
			},
			Values: &chart.Config{Raw: fmt.Sprintf(`
message: xxx
ref: %s
`, chartRef)}, //inject whatever chartRef is into the config for later inspection
		}

		return c, "", nil
	})

	rigsDir := "../../test/landscapes/multi-namespace/"
	// covers both the dir/*.yaml function as explicit files
	for _, ps := range [][]string{[]string{rigsDir}, []string{rigsDir + "hello-world.yaml", rigsDir + "secretive2.yaml", rigsDir + "secretive.yaml"}} {

		fs := NewFileStateProvider(ps, secretsMock, chartLoadMock, "pfx-", "spa")
		cs, err := fs.Components()
		require.NoError(t, err)
		require.Len(t, cs, 3)
		require.Contains(t, cs, "pfx-hello-world")
		require.Contains(t, cs, "pfx-secretive")
		require.Contains(t, cs, "pfx-secretive2")

		c0 := cs["pfx-hello-world"]
		c1 := cs["pfx-secretive"]
		c2 := cs["pfx-secretive2"]

		require.Equal(t, "hello-world:0.1.0", c0.Release.Chart)
		require.Equal(t, "0.1.0", c0.Release.Version)
		ref, err := c0.FullChartRef()
		require.NoError(t, err)
		require.Equal(t, "local/hello-world:0.1.0", ref)

		require.Equal(t, "spa", c0.Namespace)    //default
		require.Equal(t, "newnam", c1.Namespace) //overridden
		require.Equal(t, "newnam", c2.Namespace) //overridden

		require.Equal(t, "Hello, Landscaped world!", c0.Configuration["message"]) //overridden
		require.Equal(t, "xxx", c1.Configuration["message"])                      //chart default
		require.Equal(t, "local/hello-world:0.1.0", c0.Configuration["ref"])
		require.Equal(t, "local/hello-secret:1.3.37", c1.Configuration["ref"]) //unspecified in file, obtained from chart
		require.Equal(t, "local/hello-secret:0.1.0", c2.Configuration["ref"])

		require.Len(t, c1.Secrets, 2)
		require.Contains(t, c1.Secrets, "hello-name")
		require.Contains(t, c1.Secrets, "hello-age")

		require.Len(t, c1.SecretValues, 2)
		require.Equal(t, []byte("pfx-secretivenewnamh3llo-nam3"), c1.SecretValues["hello-name"])
	}
}

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
	require.Len(t, cmps, 1)
	require.Contains(t, cmps, "my-release")
	c := cmps["my-release"]
	require.Equal(t, "my-release", c.Name)
	require.Equal(t, "my-namespace", c.Namespace)
	require.Equal(t, "xxx", c.Configuration["config_a"]) // from chart
	require.Equal(t, "zzz", c.Configuration["config_b"]) // from chart but overridden in values
	require.Equal(t, "qqq", c.Configuration["config_c"]) // in values but not in chart
}
