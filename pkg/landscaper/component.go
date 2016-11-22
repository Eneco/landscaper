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
	if len(c.Secrets) > 0 {
		c.Configuration["SecretsRef"] = env.ReleaseName(c.Name)
	}
	c.Name = env.ReleaseName(c.Name)

	ss := strings.Split(c.Release.Chart, "/")
	if len(ss) != 2 {
		return fmt.Errorf("bad release.chart: `%s`, expecting `some_repo/some_name`", c.Release.Chart)
	}
	c.Release.Chart = ss[1]

	c.Configuration.SetMetadata(&Metadata{ChartRepository: ss[0], ReleaseVersion: c.Release.Version})

	return nil
}

func (cfg Configuration) HasMetadata() bool {
	_, ok := cfg[metadataKey]
	return ok
}

func (cfg Configuration) GetMetadata() (*Metadata, error) {
	val, ok := cfg[metadataKey]
	if !ok {
		return nil, fmt.Errorf("configuration has no metadata")
	}

	metadata := val.(map[string]interface{})

	return &Metadata{ReleaseVersion: metadata[metaReleaseVersion].(string), ChartRepository: metadata[metaChartRepo].(string)}, nil
}

func (cfg Configuration) SetMetadata(m *Metadata) {
	cfg[metadataKey] = map[string]interface{}{
		metaReleaseVersion: m.ReleaseVersion,
		metaChartRepo:      m.ChartRepository,
	}
}

func (c *Component) FullChartRef() (string, error) {
	m, err := c.Configuration.GetMetadata()
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%s/%s", m.ChartRepository, c.Release.Chart), nil
}
