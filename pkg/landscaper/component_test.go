package landscaper

import (
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComponent(t *testing.T) {
	cfg, err := ioutil.ReadFile("../../test/component_test_data.yaml")
	require.NoError(t, err)

	expected := &Component{
		Name: "test-component",
		Release: &Release{
			Chart:   "connectors/hdfs:0.0.7",
			Version: "1.0.0",
		},
		Configuration: Configuration{
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
		Secrets: &Secrets{
			"twitterAPIKey",
			"cloudstackKey",
		},
	}

	cmp, err := NewComponentFromYAML(cfg)
	assert.NoError(t, err)
	assert.Equal(t, expected, cmp)
}
