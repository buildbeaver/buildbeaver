package dto

import (
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
)

type CreateLogContainer struct {
	BuildID     models.BuildID
	JobID       *models.JobID
	StepID      *models.StepID
	Name        models.ResourceName
	Description string
}

type CreateLogEntry struct {
	LogContainerID models.LogDescriptorID
	Entry          models.LogEntry
}

type LogEntryKind string

type LogEntry interface {
	Kind() LogEntryKind
	SetTimestamp(t time.Time)
}

type LogEntryContainer interface {
	LogEntry
	ID() models.LogDescriptorID
}
