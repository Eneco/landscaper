package landscaper

import "gopkg.in/yaml.v2"

// Configuration contains all the values that should be applied to the component's helm package release
type Configuration map[string]interface{}

// YAML encodes the Values into a YAML string.
func (v Configuration) YAML() (string, error) {
	b, err := yaml.Marshal(v)
	return string(b), err
}
