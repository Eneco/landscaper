package landscaper

import (
	"github.com/ghodss/yaml"
	validator "gopkg.in/validator.v2"
)

// Component contains information about the release, configuration and secrets of a component
type Component struct {
	Name          string        `json:"name",validate:"nonzero,max=12"`
	Release       *Release      `json:"release",validate:"nonzero"`
	Configuration Configuration `json:"configuration"`
	Secrets       *Secrets      `json:"secrets"`
}

// NewComponentFromYAML parses a byteslice into a Component instance
func NewComponentFromYAML(content []byte) (*Component, error) {
	cmp := &Component{}
	if err := yaml.Unmarshal(content, cmp); err != nil {
		return nil, err
	}

	return cmp, nil
}

// Validate the component on required fields and correct values
func (c *Component) Validate() error {
	if err := validator.Validate(c); err != nil {
		return err
	}

	return nil
}
