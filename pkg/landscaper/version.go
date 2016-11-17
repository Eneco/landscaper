package landscaper

// plugged in during build
var (
	SemVer    = "-"
	GitCommit = "-"
	GitTag    = "-"
)

// Version contains versioning information obtained during build
type Version struct {
	SemVer    string
	GitCommit string
	GitTag    string
}

// GetVersion provides the landscaper version and build information
func GetVersion() Version {
	return Version{SemVer: SemVer, GitCommit: GitCommit, GitTag: GitTag}
}
