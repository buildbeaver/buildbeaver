package models

import (
	"strconv"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const (
	BuildResourceKind ResourceKind = "build"
)

type BuildID struct {
	ResourceID
}

func NewBuildID() BuildID {
	return BuildID{ResourceID: NewResourceID(BuildResourceKind)}
}

func BuildIDFromResourceID(id ResourceID) BuildID {
	return BuildID{ResourceID: id}
}

type BuildNumber uint64

func (m BuildNumber) String() string {
	return strconv.FormatUint(uint64(m), 10)
}

type Build struct {
	ID        BuildID      `json:"id" goqu:"skipupdate" db:"build_id"`
	Name      ResourceName `json:"name" db:"build_name"`
	RepoID    RepoID       `json:"repo_id" db:"build_repo_id"`
	CreatedAt Time         `json:"created_at" goqu:"skipupdate" db:"build_created_at"`
	UpdatedAt Time         `json:"updated_at" db:"build_updated_at"`
	DeletedAt *Time        `json:"deleted_at,omitempty" db:"build_deleted_at"`
	ETag      ETag         `json:"etag" db:"build_etag" hash:"ignore"`
	// CommitID that is being built.
	CommitID CommitID `json:"commit_id" db:"build_commit_id"`
	// LogDescriptorID points to the log for this build.
	LogDescriptorID LogDescriptorID `json:"log_descriptor_id" db:"build_log_descriptor_id"`
	// Ref is the git ref the build is for (e.g. branch or tag)
	Ref string `json:"ref" db:"build_ref"`
	// Status reflects where the build is in the queue.
	Status WorkflowStatus `json:"status" db:"build_status"`
	// Timings records the times at which the build transitioned between statuses.
	Timings WorkflowTimings `json:"timings" db:"build_timings"`
	// Error is set if the build finished with an error (or nil if the build succeeded).
	Error *Error `json:"error" db:"build_error"`
	// Opts that are applied to this build.
	Opts BuildOptions `json:"opts" db:"build_opts"`
}

func (m *Build) GetKind() ResourceKind {
	return BuildResourceKind
}

func (m *Build) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Build) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Build) GetParentID() ResourceID {
	return m.RepoID.ResourceID
}

func (m *Build) GetName() ResourceName {
	return m.Name
}

func (m *Build) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Build) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Build) GetETag() ETag {
	return m.ETag
}

func (m *Build) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Build) GetDeletedAt() *Time {
	return m.DeletedAt
}

func (m *Build) SetDeletedAt(deletedAt *Time) {
	m.DeletedAt = deletedAt
}

func (m *Build) IsUnreachable() bool {
	// Builds are unreachable after they are soft-deleted
	return m.DeletedAt != nil
}

// Validate the entire build pipeline including the job and step relationships/dependencies.
func (m *Build) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	// TODO ideally we could check the name here but there is a chicken and egg situation
	//  where the build is validated inside the config parser, but the build name (aka builder number)
	//  isn't allocated until afterwards by the queue service.
	//if err := m.Name.Validate(); err != nil {
	//	result = multierror.Append(result, err)
	//}
	if !m.RepoID.Valid() {
		result = multierror.Append(result, errors.New("error repo id must be set"))
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if !m.CommitID.Valid() {
		result = multierror.Append(result, errors.New("error commit id must be set"))
	}
	if m.Ref == "" {
		result = multierror.Append(result, errors.New("error ref must be set"))
	}
	if m.Status == "" {
		result = multierror.Append(result, errors.New("error status must be set"))
	}
	if !m.Status.Valid() {
		result = multierror.Append(result, errors.New("error status is invalid"))
	}
	return result.ErrorOrNil()
}
