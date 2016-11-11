package landscaper

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var testEnvironment = &Environment{
	Namespace:          "landscaper-testing",
	LandscapeName:      "testing",
	LandscapeDir:       "../../test",
	HelmRepositoryName: "eet",
}

func TestComponentProviderCurrent(t *testing.T) {
	cp, err := NewComponentProvider(testEnvironment)
	require.NoError(t, err)

	expected := []*Component{
		&Component{
			Name: "hdfs-test",
			Release: &Release{
				Chart:   "connector-hdfs:0.1.0",
				Version: "1.0.0",
			},
			Configuration: Configuration{
				"FlushSize":                  3.0,
				"HdfsUrl":                    "hdfs://hadoop:8020",
				"TasksMax":                   1.0,
				"LogsDir":                    "/tmp/logs",
				"FilenameOffsetZeroPadWidth": 1.0,
				"Replicas":                   1.0,
				"Image":                      "registry-github.com/eneco/connector-hdfs:latest",
				"ImagePullPolicy":            "Always",
				"GroupID":                    "hdfs-rtwind",
				"PartitionerClass":           "io.confluent.connect.hdfs.partitioner.FieldPartitioner",
				"RotateIntervalMs":           10000.0,
				"SchemaRegistryURL":          "http://schema-registry:8181",
				"ShutdownTimeoutMs":          60000.0,
				"TopicsDir":                  "/tmp/topics",
				"Name":                       "hdfs-test",
				"PartitionField":             "partition1",
				"Topics":                     "topic1,topic2",
				"RetryBackoffMs":             30000.0,
				"ImageTag":                   "latest",
				"ConnectorClass":             "io.confluent.connect.hdfs.HdfsSinkConnector",
			},
		},
	}

	components, err := cp.Current()

	require.NoError(t, err)
	assert.Equal(t, expected, components)
}

func TestComponentProviderDesired(t *testing.T) {
	cp, err := NewComponentProvider(testEnvironment)
	require.NoError(t, err)

	expected := []*Component{
		&Component{
			Name: "hdfs-test",
			Release: &Release{
				Chart:   "connector-hdfs:0.1.0",
				Version: "1.0.0",
			},
			Configuration: Configuration{
				"Topics":                     "blaat",
				"TasksMax":                   1.0,
				"HdfsUrl":                    "hdfs://hadoop:8020",
				"RetryBackoffMs":             30000.0,
				"RotateIntervalMs":           10000.0,
				"SchemaRegistryURL":          "http://schema-registry:8181",
				"Image":                      "registry-github.com/eneco/connector-hdfs:latest",
				"ConnectorClass":             "somethingwrong",
				"Name":                       "hdfs-test",
				"PartitionerClass":           "io.confluent.connect.hdfs.partitioner.FieldPartitioner",
				"ImagePullPolicy":            "Always",
				"Replicas":                   1.0,
				"LogsDir":                    "/tmp/logs",
				"TopicsDir":                  "/tmp/topics",
				"FilenameOffsetZeroPadWidth": 1.0,
				"FlushSize":                  3.0,
				"PartitionField":             "somethingelse",
				"ShutdownTimeoutMs":          60000.0,
				"ImageTag":                   "latest",
			},
		},
	}

	components, err := cp.Desired()

	fmt.Printf("%#v", components[0].Configuration)

	require.NoError(t, err)
	assert.Equal(t, expected, components)
}
