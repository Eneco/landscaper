package landscaper

// Release contains the information of a component release
type Release struct {
	Chart   string `json:"chart",validate:"nonzero"` // TODO: write a regexp validation for the chart
	Version string `json:"version",validate:"nonzero"`
}
