package models

import (
	"fmt"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	LogDescriptorResourceKind ResourceKind = "log-descriptor"
)

type LogDescriptorID struct {
	ResourceID
}

func NewLogDescriptorID() LogDescriptorID {
	return LogDescriptorID{ResourceID: NewResourceID(LogDescriptorResourceKind)}
}

func LogDescriptorIDFromResourceID(id ResourceID) LogDescriptorID {
	return LogDescriptorID{ResourceID: id}
}

// GetFileName returns a mostly-unique file name suitable for a log being downloaded.
// The filename will not include a file extension.
// The selected name is based on a shortened version of the ResourceID 'id' field which will
// make the file name relatively unique without being too long.
func (l LogDescriptorID) GetFileName() string {
	const truncateIDLength = 13
	if !l.ResourceID.Valid() || len(l.ResourceID.id) < truncateIDLength {
		return "log-INVALID-ID"
	}
	return fmt.Sprintf("log-%s", l.id[:truncateIDLength])
}

type LogDescriptor struct {
	ID          LogDescriptorID `json:"id" goqu:"skipupdate" db:"log_descriptor_id"`
	ParentLogID LogDescriptorID `json:"parent_log_id" db:"log_descriptor_parent_log_id"`
	CreatedAt   Time            `json:"created_at" goqu:"skipupdate" db:"log_descriptor_created_at"`
	UpdatedAt   Time            `json:"updated_at" db:"log_descriptor_updated_at"`
	// ResourceID is the ID of the resource that this log belongs to
	ResourceID ResourceID `json:"resource_id" db:"log_descriptor_resource_id"`
	// Sealed is set to true when the log is completed and has become immutable
	Sealed bool `json:"sealed" db:"log_descriptor_sealed"`
	// SizeBytes is calculated and set at the time the log is sealed
	SizeBytes int64 `json:"size_bytes" db:"log_descriptor_size_bytes"`
	ETag      ETag  `json:"etag" db:"log_descriptor_etag" hash:"ignore"`
}

func NewLogDescriptor(now Time, parentLogID LogDescriptorID, resourceID ResourceID) *LogDescriptor {
	return &LogDescriptor{
		ID:          NewLogDescriptorID(),
		ParentLogID: parentLogID,
		CreatedAt:   now,
		UpdatedAt:   now,
		ResourceID:  resourceID,
	}
}

func (m *LogDescriptor) GetKind() ResourceKind {
	return LogDescriptorResourceKind
}

func (m *LogDescriptor) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *LogDescriptor) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *LogDescriptor) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *LogDescriptor) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *LogDescriptor) GetETag() ETag {
	return m.ETag
}

func (m *LogDescriptor) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *LogDescriptor) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if !m.ResourceID.Valid() {
		result = multierror.Append(result, errors.New("error resource id must be set"))
	}
	return result.ErrorOrNil()
}
