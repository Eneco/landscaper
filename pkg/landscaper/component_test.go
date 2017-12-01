package landscaper

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func makeTestComp() *Component {
	return NewComponent("name", "someNameSpace", &Release{"cha", "1.1.1"}, map[string]interface{}{"config": "awesome"}, Configurations{}, Secrets{"09F911029D74E35BD84156C5635688C0"})
}

func TestComponentNew(t *testing.T) {
	cAct := NewComponent(
		"name",
		"someNameSpace",
		&Release{"cha", "1.1.1"},
		map[string]interface{}{"config": "awesome"},
		Configurations{},
		Secrets{"09F911029D74E35BD84156C5635688C0"},
	)

	cExp := &Component{
		Name:          "name",
		Namespace:     "someNameSpace",
		Release:       &Release{"cha", "1.1.1"},
		Configuration: map[string]interface{}{"config": "awesome"},
		Environments:  Configurations{},
		Secrets:       Secrets{"09F911029D74E35BD84156C5635688C0"},
		SecretValues:  SecretValues{},
	}
	cExp.Configuration[metadataKey] = map[string]interface{}{
		metaReleaseVersion: "1.1.1",
		metaChartRepo:      "",
	}

	assert.Equal(t, cExp, cAct)
}

func TestComponentValidate(t *testing.T) {
	c := makeTestComp()
	assert.NoError(t, c.Validate())

	// release can't be zero
	c = makeTestComp()
	c.Release = nil
	assert.Error(t, c.Validate())

	// name can't be longer than 12
	c = makeTestComp()
	c.Name = "way too long way too long way too long way too long way too long way too long"
	assert.Error(t, c.Validate())

	// c.Release.Chart cannot be empty
	c = makeTestComp()
	c.Release.Chart = ""
	assert.Error(t, c.Validate())

}

func TestComponentEquals(t *testing.T) {
	c0 := NewComponent("name", "default", &Release{"cha", "1.1.1"}, map[string]interface{}{"config": "awesome"}, Configurations{}, Secrets{"09F911029D74E35BD84156C5635688C0"})
	c1 := NewComponent("name", "default", &Release{"cha", "1.1.1"}, map[string]interface{}{"config": "awesome"}, Configurations{}, Secrets{"09F911029D74E35BD84156C5635688C0"})
	require.True(t, c0.Equals(c1))
	c1.Name = "other"
	require.False(t, c0.Equals(c1))
}
