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
		read: func(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			vs := SecretValues{}
			for k, s := range secretNames {
				vs[k] = []byte(componentName + namespace + strings.Replace(s, "e", "3", -1))
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
	for _, ps := range [][]string{{rigsDir}, {rigsDir + "hello-world.yaml", rigsDir + "secretive2.yaml", rigsDir + "secretive.yaml"}} {

		fs := NewFileStateProvider(ps, secretsMock, chartLoadMock, "pfx-", "spa", "", "")
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

		require.Len(t, c1.SecretNames, 2)

		require.Len(t, c1.SecretValues, 2)
		require.Equal(t, []byte("pfx-secretivenewnamh3llo-nam3"), c1.SecretValues["hello-name"])
		require.Equal(t, []byte("pfx-secretivenewnamh3llo-ag3"), c1.SecretValues["hello-age"])
	}
}

func TestOptionalVersion(t *testing.T) {
	secretsMock := SecretsProviderMock{
		read: func(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			vs := SecretValues{}
			for k, s := range secretNames {
				vs[k] = []byte(componentName + namespace + strings.Replace(s, "e", "3", -1))
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

	fs := NewFileStateProvider([]string{"../../test/landscapes/no-version/hello-world.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "", "")
	cs, err := fs.Components()
	require.NoError(t, err)
	c0 := cs["pfx-hello-world"]

	require.Equal(t, "", c0.Release.Version)
}

func TestHelmStateProviderComponents(t *testing.T) {
	helmMock := &HelmclientMock{
		listReleases: func(opts ...helm.ReleaseListOption) (*services.ListReleasesResponse, error) {
			t.Logf("listReleases %#v", opts)
			rels := &services.ListReleasesResponse{
				Releases: []*release.Release{
					{
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
		read: func(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
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

func TestMultipleEnvironments(t *testing.T) {
	secretsMock := SecretsProviderMock{
		read: func(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			vs := SecretValues{}
			for k, s := range secretNames {
				vs[k] = []byte(componentName + namespace + strings.Replace(s, "e", "3", -1))
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

	// No environment
	fs := NewFileStateProvider([]string{"../../test/landscapes/environments/hello-world.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "", "")
	cs, err := fs.Components()
	require.NoError(t, err)
	c0 := cs["pfx-hello-world"]
	require.Equal(t, "Hello, Landscaped world!", c0.Configuration["message"])
	require.Equal(t, nil, c0.Configuration["extra"])

	// Env1
	fs = NewFileStateProvider([]string{"../../test/landscapes/environments/hello-world.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "env1", "")
	cs, err = fs.Components()
	require.NoError(t, err)
	c0 = cs["pfx-hello-world"]
	require.Equal(t, "env1 overwrite", c0.Configuration["message"])
	require.Equal(t, "env1 extra", c0.Configuration["extra"])

	// Env2
	fs = NewFileStateProvider([]string{"../../test/landscapes/environments/hello-world.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "env2", "")
	cs, err = fs.Components()
	require.NoError(t, err)
	c0 = cs["pfx-hello-world"]
	require.Equal(t, "env2 overwrite", c0.Configuration["message"])
	require.Equal(t, nil, c0.Configuration["extra"])

	// Global override
	fs = NewFileStateProvider([]string{"../../test/landscapes/environments/hello-world.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "", "../../test/landscapes/environments/global-override.yaml")
	cs, err = fs.Components()
	require.NoError(t, err)
	c0 = cs["pfx-hello-world"]
	require.Equal(t, "global configuration override", c0.Configuration["message"])
	require.Equal(t, "global extra", c0.Configuration["extra"])

	// Global override + Env2
	fs = NewFileStateProvider([]string{"../../test/landscapes/environments/hello-world.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "env2", "../../test/landscapes/environments/global-override.yaml")
	cs, err = fs.Components()
	require.NoError(t, err)
	c0 = cs["pfx-hello-world"]
	require.Equal(t, "env2 overwrite", c0.Configuration["message"])
	require.Equal(t, "global extra", c0.Configuration["extra"])
}

func TestSecretLoaders(t *testing.T) {
	secretsMock := SecretsProviderMock{
		read: func(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			vs := SecretValues{}
			for k, s := range secretNames {
				vs[k] = []byte(componentName + namespace + strings.Replace(s, "e", "3", -1))
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

	// List secrets
	fs := NewFileStateProvider([]string{"../../test/landscapes/secrets/secret-list.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "", "")
	cs, err := fs.Components()
	require.NoError(t, err)
	c := cs["pfx-secret-list"]
	require.Equal(t, "These secrets are searched by name!", c.Configuration["message"])
	require.Equal(t, []byte("pfx-secret-listspalist-s3cr3t-on3"), c.SecretValues["list-secret-one"])
	require.Equal(t, []byte("pfx-secret-listspalist-s3cr3t-two"), c.SecretValues["list-secret-two"])

	// Map secrets
	fs = NewFileStateProvider([]string{"../../test/landscapes/secrets/secret-map.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "", "")
	cs, err = fs.Components()
	require.NoError(t, err)
	c = cs["pfx-secret-map"]
	require.Equal(t, "These secrets are searched by value!", c.Configuration["message"])
	require.Equal(t, []byte("pfx-secret-mapspalook-h3r3-on3"), c.SecretValues["map-secret-one"])
	require.Equal(t, []byte("pfx-secret-mapspalook-h3r3-two"), c.SecretValues["map-secret-two"])

	// No secrets
	fs = NewFileStateProvider([]string{"../../test/landscapes/secrets/secret-none.yaml"}, secretsMock, chartLoadMock, "pfx-", "spa", "", "")
	cs, err = fs.Components()
	require.NoError(t, err)
	c = cs["pfx-secret-none"]
	require.Equal(t, "There are no secrets!", c.Configuration["message"])
	require.Len(t, c.SecretValues, 0)
}

func TestHelmStateProviderSecretReading(t *testing.T) {
	helmMock := &HelmclientMock{
		listReleases: func(opts ...helm.ReleaseListOption) (*services.ListReleasesResponse, error) {
			t.Logf("listReleases %#v", opts)
			rels := &services.ListReleasesResponse{
				Releases: []*release.Release{
					{
						Name:      "my-release-no-secrets",
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
					{
						Name:      "my-release-with-secrets",
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
secretsRef: my-release-with-secrets
`, metadataKey, metaReleaseVersion, metaChartRepo)},
					},
				},
			}
			return rels, nil
		},
	}
	secretsMock := SecretsProviderMock{
		read: func(componentName, namespace string, secretNames SecretNames) (SecretValues, error) {
			t.Logf("secretsMock read %#v %#v %#v", componentName, namespace, secretNames)
			return SecretValues{componentName: []byte(componentName)}, nil
		},
	}

	hs := NewHelmStateProvider(helmMock, secretsMock, "my-prefix")
	cmps, err := hs.Components()
	require.NoError(t, err)
	require.Len(t, cmps, 2)
	require.Contains(t, cmps, "my-release-no-secrets")
	require.Contains(t, cmps, "my-release-with-secrets")

	c := cmps["my-release-no-secrets"]
	require.Equal(t, SecretValues{}, c.SecretValues)

	c = cmps["my-release-with-secrets"]
	require.Equal(t, SecretValues{"my-release-with-secrets": []byte("my-release-with-secrets")}, c.SecretValues)
}
