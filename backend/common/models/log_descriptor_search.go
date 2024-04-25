package models

type LogDescriptorSearch struct {
	Pagination
	// ParentLogID finds the log descriptor with the specified log ID and all of its children.
	ParentLogID *LogDescriptorID
}

func NewLogDescriptorSearch() *LogDescriptorSearch {
	return &LogDescriptorSearch{Pagination: Pagination{}}
}

func (m *LogDescriptorSearch) Validate() error {
	return nil
}
