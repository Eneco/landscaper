package landscaper

import validator "gopkg.in/validator.v2"

// Component contains information about the release, configuration and secrets of a component
type Component struct {
	Name          string        `json:"name",validate:"nonzero,max=12"`
	Release       *Release      `json:"release",validate:"nonzero"`
	Configuration Configuration `json:"configuration"`
	Secrets       *Secrets      `json:"secrets"`
}

// Validate the component on required fields and correct values
func (c *Component) Validate() error {
	if err := validator.Validate(c); err != nil {
		return err
	}

	return nil
}
