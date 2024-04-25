package models

type RunnerSearch struct {
	Pagination
	// LegalEntityID can be set to filter runners to those owned by a specific legal entity.
	LegalEntityID *LegalEntityID `json:"legal_entity_id"`
}

func NewRunnerSearch() *RunnerSearch {
	return &RunnerSearch{Pagination: Pagination{}}
}

func (m *RunnerSearch) Validate() error {
	return nil
}
