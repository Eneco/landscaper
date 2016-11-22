package landscaper

var (
	metadataKey        = "_landscaper_metadata"
	metaReleaseVersion = "releaseversion"
	metaChartRepo      = "chartrepository"
)

// Metadata holds landscaper metadata that is attached to a component/release through its Configuration
type Metadata struct {
	ReleaseVersion  string
	ChartRepository string
}
