package landscaper

import (
	"fmt"

	"github.com/ghodss/yaml"
)

// Configuration contains all the values that should be applied to the component's helm package release
type Configuration map[string]interface{}

type Configurations map[string]Configuration

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

func (dest Configuration) Merge(src Configuration) Configuration {
	return mergeValues(dest, src)
}

func mergeValues(dest, src Configuration) (Configuration) {
	for k, v := range src {

		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = v
			continue
		}

		// If it isn't another map, overwrite the value
		nextMap, ok := v.(map[string]interface{})
		if !ok {
			dest[k] = v
			continue
		}
		// If the key doesn't exist already, then just set the key to that value
		if _, exists := dest[k]; !exists {
			dest[k] = nextMap
			continue
		}
		// Edge case: If the key exists in the destination, but isn't a map
		destMap, isMap := dest[k].(map[string]interface{})
		// If the source map has a map for this key, prefer it
		if !isMap {
			dest[k] = v
			continue
		}
		// If we got to this point, it is a map in both, so merge them
		dest[k] = mergeValues(destMap, nextMap)
	}
	return dest
}

