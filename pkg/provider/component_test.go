package provider

import (
	"testing"

	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestReadComponentFromYAMLFilePath(t *testing.T) {
	cmp, err := ReadComponentFromYAMLFilePath("../../test/component_test_data.yaml")
	require.NoError(t, err)
	assert.NotNil(t, cmp)
}

func TestReadComponentFromCluster(t *testing.T) {
	cmp, err := ReadComponentFromCluster("traefik", &landscaper.Environment{
		Name:      "test",
		Namespace: "landscaper-testing",
	})

	assert.NoError(t, err)
	assert.Equal(t, &landscaper.Component{}, cmp)
}
