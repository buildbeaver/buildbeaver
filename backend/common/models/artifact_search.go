package models

import (
	"context"
	"fmt"

	"github.com/buildbeaver/buildbeaver/common/gerror"
)

type ArtifactSearchPaginator interface {
	HasNext() bool
	Next(ctx context.Context) ([]*Artifact, error)
}

type ArtifactSearch struct {
	Pagination
	// BuildID is the id of the build to filter artifacts to, or empty.
	BuildID BuildID `json:"build_id"`
	// Workflow is the name of the workflow that produced the artifacts being searched.
	// Nil means any workflow. Empty string means the default workflow.
	Workflow *ResourceName `json:"workflow"`
	// JobName is the name of the job that produced the artifacts being searched.
	JobName *ResourceName `json:"job_name"`
	// GroupName is the name associated with the one or more artifacts identified by an ArtifactDefinition
	// in the build config within the step being searched, or empty if the search is for all artifacts produced
	// by the step or job.
	GroupName *ResourceName `json:"group_name"`
}

func NewArtifactSearch() *ArtifactSearch {
	return &ArtifactSearch{Pagination: NewPagination(DefaultPaginationLimit, nil)}
}

func (m *ArtifactSearch) Validate() error {
	if m.JobName == nil {
		return gerror.NewErrValidationFailed("Job name must be specified")
	}
	return nil
}

type ArtifactPager struct {
	first      bool
	next       ArtifactPagerNextFn
	pagination Pagination
}

type ArtifactPagerNextFn func(ctx context.Context, pagination Pagination) ([]*Artifact, *Cursor, error)

func NewArtifactPager(initial Pagination, next ArtifactPagerNextFn) *ArtifactPager {
	return &ArtifactPager{
		first:      true,
		next:       next,
		pagination: initial,
	}
}

func (a *ArtifactPager) HasNext() bool {
	return a.first || a.pagination.Cursor != nil
}

func (a *ArtifactPager) Next(ctx context.Context) ([]*Artifact, error) {
	if !a.HasNext() {
		return nil, nil
	}
	artifacts, cursor, err := a.next(ctx, a.pagination)
	if err != nil {
		return nil, fmt.Errorf("error in next: %w", err)
	}
	a.first = false
	if cursor == nil {
		a.pagination.Cursor = nil
	} else {
		a.pagination.Cursor = cursor.Next
	}
	return artifacts, nil
}
