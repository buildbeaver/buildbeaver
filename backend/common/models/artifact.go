package models

import (
	"path/filepath"

	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

const ArtifactResourceKind ResourceKind = "artifact"

type ArtifactID struct {
	ResourceID
}

func NewArtifactID() ArtifactID {
	return ArtifactID{ResourceID: NewResourceID(ArtifactResourceKind)}
}

func ArtifactIDFromResourceID(id ResourceID) ArtifactID {
	return ArtifactID{ResourceID: id}
}

type Artifact struct {
	ID   ArtifactID `json:"id" goqu:"skipupdate" db:"artifact_id"`
	ETag ETag       `json:"etag" db:"artifact_etag" hash:"ignore"`
	// HashType is the type of hashing algorithm used to hash the data.
	HashType HashType `json:"hash_type" db:"artifact_hash_type"`
	// Hash is the hex-encoded hash of the artifact data. This may be set later if the hash is not known yet.
	Hash string `json:"hash" db:"artifact_hash"`
	// Size of the artifact file in bytes.
	Size uint64 `json:"size" db:"artifact_size"`
	// Mime type of the artifact, or empty if not known.
	Mime string `json:"mime" db:"artifact_mime"`
	// Sealed is true once the data for the artifact has successfully been uploaded and the file contents are now locked.
	// Until Sealed is true various pieces of metadata such as the file size and hash etc. will be unset.
	// NOTE: If sealed is false it doesn't necessarily mean no data has been uploaded to the blob store yet, and so
	// we must still verify that the backing data is deleted before garbage collecting unsealed artifact files.
	Sealed bool `json:"sealed" db:"artifact_sealed"`
	ArtifactData
}

type ArtifactData struct {
	Name      ResourceName `json:"name" db:"artifact_name"`
	JobID     JobID        `json:"job_id" db:"artifact_job_id"`
	CreatedAt Time         `json:"created_at" goqu:"skipupdate" db:"artifact_created_at"`
	UpdatedAt Time         `json:"updated_at" db:"artifact_updated_at"`
	// GroupName is the name associated with the one or more artifacts identified by an ArtifactDefinition in the build config.
	GroupName ResourceName `json:"group_name" db:"artifact_group_name"`
	// Path is the filesystem path that the artifact was found at, relative to the job workspace.
	Path string `json:"path" db:"artifact_path"`
}

func NewArtifactData(now Time, name ResourceName, jobID JobID, groupName ResourceName, path string) *ArtifactData {
	return &ArtifactData{
		Name:      name,
		JobID:     jobID,
		CreatedAt: now,
		UpdatedAt: now,
		GroupName: groupName,
		Path:      path,
	}
}

func (m *Artifact) GetKind() ResourceKind {
	return ArtifactResourceKind
}

func (m *Artifact) GetCreatedAt() Time {
	return m.CreatedAt
}

func (m *Artifact) GetID() ResourceID {
	return m.ID.ResourceID
}

func (m *Artifact) GetParentID() ResourceID {
	return m.JobID.ResourceID
}

func (m *Artifact) GetName() ResourceName {
	return m.Name
}

func (m *Artifact) GetUpdatedAt() Time {
	return m.UpdatedAt
}

func (m *Artifact) SetUpdatedAt(t Time) {
	m.UpdatedAt = t
}

func (m *Artifact) GetETag() ETag {
	return m.ETag
}

func (m *Artifact) SetETag(eTag ETag) {
	m.ETag = eTag
}

func (m *Artifact) Validate() error {
	var result *multierror.Error
	if !m.ID.Valid() {
		result = multierror.Append(result, errors.New("error id must be set"))
	}
	if !m.JobID.Valid() {
		result = multierror.Append(result, errors.New("error parent job id must be set"))
	}
	if err := m.Name.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.CreatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error created at must be set"))
	}
	if m.UpdatedAt.IsZero() {
		result = multierror.Append(result, errors.New("error updated at must be set"))
	}
	if err := m.GroupName.Validate(); err != nil {
		result = multierror.Append(result, err)
	}
	if m.Path == "" {
		result = multierror.Append(result, errors.New("error path must be set"))
	}
	if filepath.IsAbs(m.Path) {
		result = multierror.Append(result, errors.New("error path must be a relative path"))
	}
	return result.ErrorOrNil()
}
