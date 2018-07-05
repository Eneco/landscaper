package landscaper

import (
	"testing"

	"k8s.io/helm/pkg/helm"
	"k8s.io/helm/pkg/proto/hapi/chart"
	"k8s.io/helm/pkg/proto/hapi/services"

	"errors"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	waitTimeout = 60
)

var (
	disabledStages = make([]string, 0)
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

func TestExecutorApply(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"

	nu := newTestComponent("new-one")
	nu.Namespace = "recognizable-new-one"
	rem := newTestComponent("busted-one")
	up := newTestComponent("updated-one")
	updiff := newTestComponent("updated-one")
	updiff.Configuration["FlushSize"] = 4

	des := Components{nu.Name: nu, updiff.Name: updiff}
	cur := Components{rem.Name: rem, up.Name: up}

	helmMock := &HelmclientMock{
		installRelease: func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
			t.Logf("installRelease %#v %#v %#v", chStr, namespace, opts)
			require.Equal(t, namespace, "recognizable-new-one") // the name is hidden in the opts we cannot inspect
			return nil, nil
		},
		deleteRelease: func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
			t.Logf("deleteRelease %#v", rlsName)
			require.Equal(t, rlsName, "busted-one")
			return nil, nil
		},
		updateRelease: func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
			t.Logf("updateRelease %#v %#v %#v", rlsName, chStr, opts)
			require.Equal(t, rlsName, "updated-one")
			return nil, nil
		}}
	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		return nil, chartPath, nil
	})
	secretsMock := SecretsProviderMock{
		write: func(componentName, namespace string, values SecretValues) error {
			return nil
		},
		delete: func(componentName, namespace string) error {
			return nil
		},
	}

	err := NewExecutor(helmMock, chartLoadMock, secretsMock, false, false, waitTimeout, disabledStages).Apply(des, cur)
	require.NoError(t, err)

}

func TestExecutorApplyWithForcedUpdatesAndDeleteCreateDisable(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"

	nu := newTestComponent("new-one")
	nu.Namespace = "recognizable-new-one"
	rem := newTestComponent("busted-one")
	up := newTestComponent("updated-one")
	updiff := newTestComponent("updated-one")
	updiff.Configuration["FlushSize"] = 4
	updiff.SecretNames = SecretNames{"newSecret": "somethingNew"}

	updiff.SecretValues = SecretValues{
		"newSecret": []byte("somethingNew"),
	}

	des := Components{nu.Name: nu, updiff.Name: updiff}
	cur := Components{rem.Name: rem, up.Name: up}

	helmMock := &HelmclientMock{
		installRelease: func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
			t.Logf("installRelease %#v %#v %#v", chStr, namespace, opts)
			require.Equal(t, namespace, "recognizable-new-one") // the name is hidden in the opts we cannot inspect
			return nil, nil
		},
		deleteRelease: func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
			t.Logf("deleteRelease %#v", rlsName)
			require.Equal(t, rlsName, "busted-one")
			return nil, nil
		},
		updateRelease: func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
			t.Logf("updateRelease %#v %#v %#v", rlsName, chStr, opts)
			require.Equal(t, rlsName, "updated-one")
			return nil, nil
		}}
	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		return nil, chartPath, nil
	})
	secretsMock := SecretsProviderMock{
		write: func(componentName, namespace string, values SecretValues) error {
			return nil
		},
		delete: func(componentName, namespace string) error {
			return nil
		},
	}

	createDeleteDisabled := []string{"create", "delete"}

	err := NewExecutor(helmMock, chartLoadMock, secretsMock, false, false, waitTimeout, createDeleteDisabled).Apply(des, cur)
	require.NoError(t, err)

}

func TestExecutorCreate(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"
	nameSpace := "spacename"

	comp := newTestComponent("z")

	comp.Namespace = nameSpace
	helmMock := &HelmclientMock{installRelease: func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
		t.Logf("installRelease %#v %#v %#v", chStr, namespace, opts)
		require.Equal(t, chartPath, chStr)
		require.Equal(t, nameSpace, namespace)
		return nil, nil
	}}
	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, "repo/"+comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})
	secretsMock := SecretsProviderMock{write: func(componentName, namespace string, values SecretValues) error {
		t.Logf("secretsMock write %#v %#v %#v", componentName, namespace, values)
		require.Equal(t, comp.Name, componentName)
		require.Equal(t, comp.SecretValues, values)
		return nil
	}}

	err := NewExecutor(helmMock, chartLoadMock, secretsMock, false, false, waitTimeout, disabledStages).CreateComponent(comp)
	require.NoError(t, err)
}

func TestExecutorUpdate(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"

	comp := newTestComponent("y")

	comp.Configuration["Name"] = comp.Name

	helmMock := &HelmclientMock{updateRelease: func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
		t.Logf("updateRelease %#v %#v %#v", rlsName, chStr, opts)
		require.Equal(t, comp.Name, rlsName)
		require.Equal(t, chartPath, chStr)
		return nil, nil
	}}
	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, "repo/"+comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})
	secretsMock := SecretsProviderMock{
		write: func(componentName, namespace string, values SecretValues) error {
			require.Equal(t, comp.Name, componentName)
			require.Equal(t, comp.SecretValues, values)
			return nil
		},
		delete: func(componentName, namespace string) error {
			require.Equal(t, comp.Name, componentName)
			return nil
		},
	}

	err := NewExecutor(helmMock, chartLoadMock, secretsMock, false, false, waitTimeout, disabledStages).UpdateComponent(comp)
	require.NoError(t, err)
}

func TestExecutorDelete(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"

	comp := newTestComponent("x")

	comp.Configuration["Name"] = comp.Name

	helmMock := &HelmclientMock{deleteRelease: func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
		t.Logf("deleteRelease %#v", rlsName)
		require.Equal(t, comp.Name, rlsName)
		return nil, nil
	}}
	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		require.Equal(t, comp.Release.Chart, chartRef)
		return nil, chartPath, nil
	})
	secretsMock := SecretsProviderMock{delete: func(componentName, namespace string) error {
		require.Equal(t, comp.Name, componentName)
		return nil
	}}

	err := NewExecutor(helmMock, chartLoadMock, secretsMock, false, false, waitTimeout, disabledStages).DeleteComponent(comp)
	require.NoError(t, err)
}

func TestIsOnlySecretValueDiff(t *testing.T) {
	a := *newTestComponent("a")
	require.False(t, isOnlySecretValueDiff(a, a), "Identical components")

	b := *newTestComponent("a")
	b.Name = b.Name + "X"
	require.False(t, isOnlySecretValueDiff(a, b), "Components different on non-secretvals")

	c := *newTestComponent("a")
	c.SecretValues["x"] = []byte("y")
	require.True(t, isOnlySecretValueDiff(a, c), "Components different only on secretvals")
}

func TestIntegrateForcedUpdates(t *testing.T) {
	c := newTestComponent("C")
	u := newTestComponent("U")
	d := newTestComponent("D")
	f := newTestComponent("F")

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

func TestExecutorApplyWithDisabledStages(t *testing.T) {
	chartPath := "/opt/store/whatever/path/"

	nu := newTestComponent("new-one")
	nu.Namespace = "recognizable-new-one"
	rem := newTestComponent("busted-one")
	up := newTestComponent("updated-one")
	updiff := newTestComponent("updated-one")
	updiff.Configuration["FlushSize"] = 4

	des := Components{nu.Name: nu, updiff.Name: updiff}
	cur := Components{rem.Name: rem, up.Name: up}

	helmMock := &HelmclientMock{
		installRelease: func(chStr string, namespace string, opts ...helm.InstallOption) (*services.InstallReleaseResponse, error) {
			t.Logf("installRelease %#v %#v %#v", chStr, namespace, opts)
			require.Equal(t, namespace, "recognizable-new-one") // the name is hidden in the opts we cannot inspect
			return nil, errors.New("Shouldn't be called here")
		},
		deleteRelease: func(rlsName string, opts ...helm.DeleteOption) (*services.UninstallReleaseResponse, error) {
			t.Logf("deleteRelease %#v", rlsName)
			require.Equal(t, rlsName, "busted-one")
			return nil, errors.New("Shouldn't be called here")
		},
		updateRelease: func(rlsName string, chStr string, opts ...helm.UpdateOption) (*services.UpdateReleaseResponse, error) {
			t.Logf("updateRelease %#v %#v %#v", rlsName, chStr, opts)
			require.Equal(t, rlsName, "updated-one")
			return nil, errors.New("Shouldn't be called here")
		}}
	chartLoadMock := MockChartLoader(func(chartRef string) (*chart.Chart, string, error) {
		t.Logf("MockChartLoader %#v", chartRef)
		return nil, chartPath, nil
	})
	secretsMock := SecretsProviderMock{
		write: func(componentName, namespace string, values SecretValues) error {
			return nil
		},
		delete: func(componentName, namespace string) error {
			return nil
		},
	}

	err := NewExecutor(helmMock, chartLoadMock, secretsMock, false, false, waitTimeout, []string{"delete", "create", "update"}).Apply(des, cur)
	require.NoError(t, err)
}

func newTestComponent(name string) *Component {
	cmp := NewComponent(
		name,
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
		Configurations{},
		SecretNames{"TestSecret1": "TestSecret1", "TestSecret2": "TestSecret2"},
	)

	cmp.SecretValues = SecretValues{
		"TestSecret1": []byte("secret value 1"),
		"TestSecret2": []byte("secret value 2"),
	}

	cmp.Configuration.SetMetadata(&Metadata{ChartRepository: "repo", ReleaseVersion: "1.0.0"})

	return cmp
}
