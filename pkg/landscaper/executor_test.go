package landscaper

import (
	"testing"

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
	exec, err := NewExecutor(testEnvironment)
	require.NoError(t, err)

	err = exec.CreateComponent(NewComponent(
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
		&Secrets{},
	))
	require.NoError(t, err)
}

func TestExecutorUpdate(t *testing.T) {
	exec, err := NewExecutor(testEnvironment)
	require.NoError(t, err)

	err = exec.UpdateComponent(NewComponent(
		"create-test",
		&Release{
			Chart:   "connector-hdfs:0.1.0",
			Version: "1.1.0",
		},
		Configuration{
			"GroupID":                    "hdfs-rtwind",
			"HdfsUrl":                    "hdfs://hadoop:8020",
			"PartitionField":             "partition1",
			"TasksMax":                   1,
			"Topics":                     "topic3,topic4",
			"FlushSize":                  3,
			"FilenameOffsetZeroPadWidth": 1,
		},
		&Secrets{},
	))
	require.NoError(t, err)
}

func TestExecutorDelete(t *testing.T) {
	exec, err := NewExecutor(testEnvironment)
	require.NoError(t, err)

	err = exec.DeleteComponent(NewComponent(
		"create-test",
		&Release{},
		Configuration{},
		&Secrets{},
	))
	require.NoError(t, err)
}
