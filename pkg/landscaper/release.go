package landscaper

import "fmt"

// Release contains the information of a component release
type Release struct {
	Chart   string `json:"chart" validate:"nonzero"` // TODO: write a regexp validation for the chart
	Version string `json:"version" validate:"nonzero"`
	repo    string
}

func (r *Release) fullChartRef() string {
	return fmt.Sprintf("%s/%s", r.repo, r.Chart)
}
