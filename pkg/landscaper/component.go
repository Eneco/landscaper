package landscaper

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/validator.v2"
)

// Component contains information about the release, configuration and secrets of a component
type Component struct {
	Name          string        `json:"name" validate:"nonzero,max=51"`
	Release       *Release      `json:"release" validate:"nonzero"`
	Configuration Configuration `json:"configuration"`
	Secrets       Secrets       `json:"secrets"`
	SecretValues  SecretValues  `json:"-"`
}

// NewComponent creates a Component and adds Name to the configuration
func NewComponent(name string, release *Release, cfg Configuration, secrets Secrets) *Component {
	cmp := &Component{
		Name:          name,
		Release:       release,
		Configuration: cfg,
		Secrets:       secrets,
		SecretValues:  SecretValues{},
	}

	if cmp.Configuration == nil {
		cmp.Configuration = Configuration{}
	}

	if cmp.Secrets == nil {
		cmp.Secrets = Secrets{}
	}

	cmp.Configuration[metadataKey] = Metadata{ReleaseVersion: cmp.Release.Version}

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

func (c *Component) normalizeFromFile(env *Environment) error {
	c.Configuration["Name"] = c.Name
	c.Configuration["SecretsRef"] = env.ReleaseName(c.Name)
	c.Name = env.ReleaseName(c.Name)

	// releases from file contain repo as part of the chartname
	if !strings.Contains(c.Release.Chart, "/") {
		if c.Release.repo == "" {
			return fmt.Errorf("bad") //TODO
		}
		return nil
	}

	ss := strings.Split(c.Release.Chart, "/")
	if len(ss) != 2 {
		return fmt.Errorf("bad release.chart: `%s`", c.Release.Chart)
	}
	c.Release.repo = ss[0]
	c.Release.Chart = ss[1]

	m := c.Configuration[metadataKey].(Metadata)
	m.ChartRepository = c.Release.repo
	c.Configuration[metadataKey] = m
	return nil
}

func (c *Component) normalizeFromHelm(repo string) {
	// releases from helm lack the repo name
	c.Release.repo = repo

	c.Secrets = Secrets{}
	for key := range c.SecretValues {
		c.Secrets = append(c.Secrets, key)
	}
}
