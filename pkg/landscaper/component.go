package landscaper

import (
	"fmt"
	"reflect"

	"gopkg.in/validator.v2"
)

// Component contains information about the release, configuration and secrets of a component
type Component struct {
	Name          string        `json:"name" validate:"nonzero,max=51"`
	Release       *Release      `json:"release" validate:"nonzero"`
	Configuration Configuration `json:"configuration"`
	Secrets       Secrets       `json:"secrets"`
}

// NewComponent creates a Component and adds Name to the configuration
func NewComponent(name string, release *Release, cfg Configuration, secrets Secrets) *Component {
	cmp := &Component{
		Name:          name,
		Release:       release,
		Configuration: cfg,
		Secrets:       secrets,
	}

	cmp.Configuration[metadataKey] = map[string]interface{}{
		releaseVersionKey: cmp.Release.Version,
		landscaperTagKey:  true,
	}

	return cmp
}

// Validate the component on required fields and correct values
func (c *Component) Validate() error {
	return validator.Validate(c)
}

// Equals checks if this component's values are equal to another
func (c *Component) Equals(other *Component) bool {
	return reflect.DeepEqual(c, other)
}

// validateComponents validates the individual components as well as duplicate names in the total collection
func validateComponents(cs []*Component) error {
	// are the individual components okay?
	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return err
		}
	}

	// is the collection as a whole okay: no dup names?
	cMap := make(map[string]*Component)

	for _, c := range cs {
		if _, ok := cMap[c.Name]; ok {
			return fmt.Errorf("duplicate component name `%s`", c.Name)
		}
		cMap[c.Name] = c
	}

	return nil
}
