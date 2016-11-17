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
	current := []*Component{
		&Component{Name: "cmpA"},
		&Component{Name: "cmpB", Release: &Release{Chart: "chart1"}},
		&Component{Name: "cmpC"},
	}

	desired := []*Component{
		&Component{Name: "cmpD"},
		&Component{Name: "cmpB", Release: &Release{Chart: "chart2"}},
		&Component{Name: "cmpC"},
	}

	actualC, actualU, actualD := diff(desired, current)

	expectedC := []*Component{&Component{Name: "cmpD"}}
	expectedU := []*Component{&Component{Name: "cmpB", Release: &Release{Chart: "chart2"}}}
	expectedD := []*Component{&Component{Name: "cmpA"}}

	assert.Equal(t, expectedC, actualC)
	assert.Equal(t, expectedU, actualU)
	assert.Equal(t, expectedD, actualD)
}

func TestExecutorCreate(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"
	nameSpace := "spacename"

	comp := newTestComponent()
	env := newTestEnvironment()

	env.Namespace = nameSpace
	env.HelmClient = &HelmclientMock{installRelease: func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
		t.Logf("installRelease %#v %#v %#v", chStr, namespace, opts)
		require.Equal(t, chartPath, chStr)
		require.Equal(t, nameSpace, namespace)
		return nil, nil
	}}
	env.ChartLoader = MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, chartRef, env.HelmRepositoryName+"/"+comp.Release.Chart)
		return nil, chartPath, nil
	})

	exec, err := NewExecutor(env)
	require.NoError(t, err)

	err = exec.CreateComponent(comp)
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
	env.HelmClient = &HelmclientMock{updateRelease: func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
		t.Logf("updateRelease %#v %#v %#v", rlsName, chStr, opts)
		require.Equal(t, rlsName, comp.Name)
		require.Equal(t, chartPath, chStr)
		return nil, nil
	}}
	env.ChartLoader = MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, chartRef, env.HelmRepositoryName+"/"+comp.Release.Chart)
		return nil, chartPath, nil
	})

	exec, err := NewExecutor(env)
	require.NoError(t, err)

	err = exec.UpdateComponent(comp)
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
	env.HelmClient = &HelmclientMock{deleteRelease: func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
		t.Logf("deleteRelease %#v", rlsName)
		require.Equal(t, comp.Name, rlsName)
		return nil, nil
	}}
	env.ChartLoader = MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, env.HelmRepositoryName+"/"+comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})

	exec, err := NewExecutor(env)
	require.NoError(t, err)

	err = exec.DeleteComponent(comp)
	require.NoError(t, err)
}

func newTestComponent() *Component {
	return NewComponent(
		"create-test",
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
}

func newTestEnvironment() *Environment {
	return &Environment{
		Namespace:          "landscaper-testing",
		LandscapeName:      "testing",
		LandscapeDir:       "../../test",
		HelmRepositoryName: "eet",
	}
}
