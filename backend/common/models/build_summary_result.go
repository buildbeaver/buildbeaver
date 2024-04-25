package models

// BuildSummaryResult represents the content for a Build Summary call against the API.
type BuildSummaryResult struct {
	Running   []*BuildSearchResult `json:"running"`
	Upcoming  []*BuildSearchResult `json:"upcoming"`
	Completed []*BuildSearchResult `json:"completed"`
}
