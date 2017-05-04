package landscaper

import (
	"fmt"

	"gopkg.in/yaml.v2"
)

// Configuration contains all the values that should be applied to the component's helm package release
type Configuration map[string]interface{}

// YAML encodes the Values into a YAML string.
func (cfg Configuration) YAML() (string, error) {
	b, err := yaml.Marshal(cfg)
	return string(b), err
}

// HasMetadata returns true if the config contains a landscaper metadata structure
func (cfg Configuration) HasMetadata() bool {
	_, ok := cfg[metadataKey]
	return ok
}

// GetMetadata returns a Metadata if present
func (cfg Configuration) GetMetadata() (*Metadata, error) {
	val, ok := cfg[metadataKey]
	if !ok {
		return nil, fmt.Errorf("configuration has no metadata")
	}

	metadata := val.(map[string]interface{})

	return &Metadata{ReleaseVersion: metadata[metaReleaseVersion].(string), ChartRepository: metadata[metaChartRepo].(string)}, nil
}

// SetMetadata sets the provided Metadata
func (cfg Configuration) SetMetadata(m *Metadata) {
	cfg[metadataKey] = map[string]interface{}{
		metaReleaseVersion: m.ReleaseVersion,
		metaChartRepo:      m.ChartRepository,
	}
}
