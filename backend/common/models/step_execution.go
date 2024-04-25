package models

import (
	"strings"

	"github.com/pkg/errors"
)

const (
	StepExecutionSequential StepExecution = "sequential"
	StepExecutionParallel   StepExecution = "parallel"
)

type StepExecution string

func (m *StepExecution) Scan(src interface{}) error {
	if src == nil {
		*m = StepExecutionSequential
		return nil
	}
	t, ok := src.(string)
	if !ok {
		return errors.Errorf("error expected string but found: %T", src)
	}
	switch strings.ToLower(t) {
	case "", string(StepExecutionSequential):
		*m = StepExecutionSequential
	case string(StepExecutionParallel):
		*m = StepExecutionParallel
	default:
		return errors.Errorf("error unknown step execution: %s", t)
	}
	return nil
}

func (m StepExecution) Valid() bool {
	return m == StepExecutionSequential || m == StepExecutionParallel
}

func (m StepExecution) String() string {
	return string(m)
}
