package landscaper

var (
	metadataKey        = "_landscaper_metadata"
	metaReleaseVersion = "releaseversion"
	metaChartRepo      = "chartrepository"
)

type Metadata struct {
	ReleaseVersion  string
	ChartRepository string
}
