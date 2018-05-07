package landscaper

import (
	"testing"
	"github.com/stretchr/testify/assert"
	"gopkg.in/jarcoal/httpmock.v1"
	"io/ioutil"
)

func TestLoadLocalCharts(t *testing.T) {

	localCharts := NewLocalCharts("../../test/helm/home")

	_, _, err := localCharts.Load("hello")
	assert.NotNil(t, err)

	httpmock.Activate()
	defer httpmock.DeactivateAndReset()
	resp, _ := ioutil.ReadFile("../../test/helm/home/repository/cache/archive/hello-cron-0.1.0.tgz")
	httpmock.RegisterResponder("GET", "http://example.com/hello-cron-0.1.0.tgz",
		httpmock.NewBytesResponder(200, resp))

	chart, _, err := localCharts.Load("landscapeTest/hello-cron")
	assert.Nil(t, err)
	assert.Equal(t, "hello-cron", chart.Metadata.Name)
}

