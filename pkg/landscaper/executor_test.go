package landscaper

import (
	"testing"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/services"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExecutorDiff(t *testing.T) {
	current := Components{
		"cmpA": &Component{Name: "cmpA"},
		"cmpB": &Component{Name: "cmpB", Release: &Release{Chart: "chart1"}},
		"cmpC": &Component{Name: "cmpC"},
	}

	desired := Components{
		"cmpD": &Component{Name: "cmpD"},
		"cmpB": &Component{Name: "cmpB", Release: &Release{Chart: "chart2"}},
		"cmpC": &Component{Name: "cmpC"},
	}

	actualC, actualU, actualD := diff(desired, current)

	expectedC := Components{"cmpD": &Component{Name: "cmpD"}}
	expectedU := Components{"cmpB": &Component{Name: "cmpB", Release: &Release{Chart: "chart2"}}}
	expectedD := Components{"cmpA": &Component{Name: "cmpA"}}

	assert.Equal(t, expectedC, actualC)
	assert.Equal(t, expectedU, actualU)
	assert.Equal(t, expectedD, actualD)
}

func TestExecutorCreate(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"
	nameSpace := "spacename"

	comp := newTestComponent()
	env := newTestEnvironment()

	comp.Namespace = nameSpace
	env.helmClient = &HelmclientMock{installRelease: func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
		t.Logf("installRelease %#v %#v %#v", chStr, namespace, opts)
		require.Equal(t, chartPath, chStr)
		require.Equal(t, nameSpace, namespace)
		return nil, nil
	}}
	env.ChartLoader = MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, "repo/"+comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})

	err := NewExecutor(env, SecretsProviderMock{write: func(componentName, namespace string, values SecretValues) error {
		require.Equal(t, comp.Name, componentName)
		require.Equal(t, comp.SecretValues, values)
		return nil
	}}).CreateComponent(comp)
	require.NoError(t, err)
}

func TestExecutorUpdate(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"
	nameSpace := "spacename"

	comp := newTestComponent()
	env := newTestEnvironment()

	comp.Configuration["Name"] = comp.Name
	comp.Name = env.ReleaseName(comp.Name)

	env.Namespace = nameSpace
	env.helmClient = &HelmclientMock{updateRelease: func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
		t.Logf("updateRelease %#v %#v %#v", rlsName, chStr, opts)
		require.Equal(t, comp.Name, rlsName)
		require.Equal(t, chartPath, chStr)
		return nil, nil
	}}
	env.ChartLoader = MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, "repo/"+comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})

	err := NewExecutor(env, SecretsProviderMock{
		write: func(componentName, namespace string, values SecretValues) error {
			require.Equal(t, comp.Name, componentName)
			require.Equal(t, comp.SecretValues, values)
			return nil
		},
		delete: func(componentName, namespace string) error {
			require.Equal(t, comp.Name, componentName)
			return nil
		},
	}).UpdateComponent(comp)
	require.NoError(t, err)
}

func TestExecutorDelete(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"
	nameSpace := "spacename"

	comp := newTestComponent()
	env := newTestEnvironment()

	comp.Configuration["Name"] = comp.Name
	comp.Name = env.ReleaseName(comp.Name)

	env.Namespace = nameSpace
	env.helmClient = &HelmclientMock{deleteRelease: func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
		t.Logf("deleteRelease %#v", rlsName)
		require.Equal(t, comp.Name, rlsName)
		return nil, nil
	}}
	env.ChartLoader = MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})

	err := NewExecutor(env, SecretsProviderMock{delete: func(componentName, namespace string) error {
		require.Equal(t, comp.Name, componentName)
		return nil
	}}).DeleteComponent(comp)
	require.NoError(t, err)
}

func TestIsOnlySecretValueDiff(t *testing.T) {
	a := *newTestComponent()
	require.False(t, isOnlySecretValueDiff(a, a), "Identical components")

	b := *newTestComponent()
	b.Name = b.Name + "X"
	require.False(t, isOnlySecretValueDiff(a, b), "Components different on non-secretvals")

	c := *newTestComponent()
	c.SecretValues["x"] = []byte("y")
	require.True(t, isOnlySecretValueDiff(a, c), "Components different only on secretvals")
}

func TestIntegrateForcedUpdates(t *testing.T) {
	c := newTestComponent()
	u := newTestComponent()
	d := newTestComponent()
	f := newTestComponent()
	c.Name = "C"
	u.Name = "U"
	d.Name = "D"
	f.Name = "F"

	current := Components{u.Name: u, f.Name: f, d.Name: d}

	create := Components{c.Name: c}
	update := Components{u.Name: u, f.Name: f}
	delete := Components{d.Name: d}

	needForcedUpdate := map[string]bool{"F": true}

	create, update, delete = integrateForcedUpdates(current, create, update, delete, needForcedUpdate)

	require.Equal(t, Components{c.Name: c, f.Name: f}, create)
	require.Equal(t, Components{u.Name: u}, update)
	require.Equal(t, Components{d.Name: d, f.Name: f}, delete)
}

func newTestComponent() *Component {
	cmp := NewComponent(
		"create-test",
		"myNameSpace",
		&Release{
			Chart:   "connector-hdfs:0.1.0",
			Version: "1.0.0",
		},
		Configuration{
			"GroupID":                    "hdfs-rtwind",
			"HdfsUrl":                    "hdfs://hadoop:8020",
			"PartitionField":             "partition1",
			"TasksMax":                   1,
			"Topics":                     "topic1,topic2",
			"FlushSize":                  3,
			"FilenameOffsetZeroPadWidth": 1,
		},
		Secrets{},
	)

	cmp.SecretValues = SecretValues{
		"TestSecret1": []byte("secret value 1"),
		"TestSecret2": []byte("secret value 2"),
	}

	cmp.Configuration.SetMetadata(&Metadata{ChartRepository: "repo", ReleaseVersion: "1.0.0"})

	return cmp
}

func newTestEnvironment() *Environment {
	return &Environment{
		Namespace:         "landscaper-testing",
		ReleaseNamePrefix: "testing",
		LandscapeDir:      "../../test",
	}
}
