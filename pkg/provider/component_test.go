package provider

import (
	"io/ioutil"
	"testing"

	"github.com/eneco/landscaper/pkg/landscaper"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComponent(t *testing.T) {
	cfg, err := ioutil.ReadFile("../../test/component_test_data.yaml")
	require.NoError(t, err)

	expected := &landscaper.Component{
		Name: "test-component",
		Release: &landscaper.Release{
			Chart:   "connectors/hdfs:0.0.7",
			Version: "1.0.0",
		},
		Configuration: landscaper.Configuration{
			"filenameOffsetZeroPadWidth": 1.0,
			"shutdownTimeoutMs":          60000.0,
			"tasksMax":                   1.0,
			"groupID":                    "hdfs-rtwind",
			"name":                       "hdfs-rtwind",
			"partitionerClass":           "io.confluent.connect.hdfs.partitioner.FieldPartitioner",
			"retryBackoffMs":             30000.0,
			"topicsDir":                  "/tmp/topics",
			"flushSize":                  3.0,
			"logsDir":                    "/tmp/logs",
			"rotateIntervalMs":           10000.0,
			"hdfsUrl":                    "hdfs://hadoop:8020",
			"partitionField":             "partition1",
			"schemaRegistryURL":          "http://schema-registry:8181",
			"topics":                     "topic1,topic2",
		},
		Secrets: &landscaper.Secrets{
			"twitterAPIKey",
			"cloudstackKey",
		},
	}

	cmp, err := NewComponentFromYAML(cfg)
	assert.NoError(t, err)
	assert.Equal(t, expected, cmp)
}

func TestReadComponentFromYAMLFilePath(t *testing.T) {
	cmp, err := ReadComponentFromYAMLFilePath("../../test/component_test_data.yaml")
	require.NoError(t, err)
	assert.NotNil(t, cmp)
}

func TestCoalesceComponent(t *testing.T) {
	cmp, err := ReadComponentFromYAMLFilePath("../../test/component_test_data.yaml")
	require.NoError(t, err)

	err = CoalesceComponent(cmp, "eet")
	require.NoError(t, err)

	assert.Equal(t, cmp.Configuration["PartitionerClass"], "io.confluent.connect.hdfs.partitioner.FieldPartitioner")
}

func TestListComponentsFromCluster(t *testing.T) {
	components, err := ListComponentsFromCluster(&landscaper.Environment{
		Name:      "test",
		Namespace: "landscaper-testing",
	})

	assert.NoError(t, err)
	assert.NotNil(t, components[0])
}

func TestListComponentsFromFolder(t *testing.T) {
	components, err := ListComponentsFromFolder("../../test", "eet")

	assert.NoError(t, err)
	assert.NotNil(t, components[0])
}

func TestReadComponentFromCluster(t *testing.T) {
	expected, err := ReadComponentFromYAMLFilePath("../../test/component_test_data.yaml")
	require.NoError(t, err)

	cmp, err := ReadComponentFromCluster("traefik", &landscaper.Environment{
		Name:      "test",
		Namespace: "landscaper-testing",
	})

	assert.NoError(t, err)
	assert.Equal(t, expected, cmp)
}
