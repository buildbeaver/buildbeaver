package models

import "github.com/buildbeaver/buildbeaver/common/gerror"

type LogSearch struct {
	StartSeqNo *int
	Plaintext  *bool
	Expand     *bool
}

func NewLogSearch() *LogSearch {
	return &LogSearch{}
}

func (m *LogSearch) Validate() error {
	if (m.Expand != nil && *m.Expand) && (m.StartSeqNo != nil && *m.StartSeqNo != 0) {
		return gerror.NewErrValidationFailed("expand and start seq no cannot be specified together")
	}
	return nil
}
