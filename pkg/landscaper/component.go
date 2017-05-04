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
	Namespace     string        `json:"namespace"`
	Release       *Release      `json:"release" validate:"nonzero"`
	Configuration Configuration `json:"configuration"`
	Secrets       Secrets       `json:"secrets"`
	SecretValues  SecretValues  `json:"-"`
}

// Components is a collection of uniquely named Component objects
type Components map[string]*Component

// NewComponent creates a Component and adds Name to the configuration
func NewComponent(name string, namespace string, release *Release, cfg Configuration, secrets Secrets) *Component {
	cmp := &Component{
		Name:          name,
		Release:       release,
		Configuration: cfg,
		Secrets:       secrets,
		SecretValues:  SecretValues{},
		Namespace:     namespace,
	}

	if cmp.Configuration == nil {
		cmp.Configuration = Configuration{}
	}

	if cmp.Secrets == nil {
		cmp.Secrets = Secrets{}
	}

	m := &Metadata{}
	if cmp.Configuration.HasMetadata() {
		m, _ = cmp.Configuration.GetMetadata()
	}
	m.ReleaseVersion = cmp.Release.Version
	cmp.Configuration.SetMetadata(m)

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
func validateComponents(cs Components) error {
	// are the individual components okay?
	for _, c := range cs {
		if err := c.Validate(); err != nil {
			return err
		}
	}

	return nil
}

func (c *Component) normalizeFromFile(releaseNamePrefix, namespace string) error {
	c.Configuration["Name"] = c.Name
	c.Name = releaseNamePrefix + strings.ToLower(c.Name)
	if len(c.Secrets) > 0 {
		c.Configuration["secretsRef"] = c.Name
	}

	ss := strings.Split(c.Release.Chart, "/")
	if len(ss) != 2 {
		return fmt.Errorf("bad release.chart: `%s`, expecting `some_repo/some_name`", c.Release.Chart)
	}
	c.Release.Chart = ss[1]

	c.Configuration.SetMetadata(&Metadata{ChartRepository: ss[0], ReleaseVersion: c.Release.Version})

	if c.Namespace == "" {
		c.Namespace = namespace
	}

	return nil
}

// FullChartRef provides a chart references like "myRepo/chartName"
func (c *Component) FullChartRef() (string, error) {
	m, err := c.Configuration.GetMetadata()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", m.ChartRepository, c.Release.Chart), nil
}
