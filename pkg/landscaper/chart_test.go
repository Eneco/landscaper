package landscaper

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/helm/pkg/repo/repotest"
)

func TestLoadLocalCharts(t *testing.T) {

	tmp := filepath.Join(os.TempDir(), "landscaper", "landscapeTest")
	defer os.RemoveAll(tmp)

	localCharts := NewLocalCharts("testdata/helmhome")

	_, _, err := localCharts.Load("hello")
	assert.NotNil(t, err)

	srv := repotest.NewServer(tmp)
	defer srv.Stop()

	if _, err := srv.CopyCharts("testdata/*.tgz*"); err != nil {
		t.Error(err)
		return
	}

	chart, _, err := localCharts.Load("landscapeTest/hello-cron")
	assert.Nil(t, err)
	assert.Equal(t, "hello-cron", chart.Metadata.Name)
}
