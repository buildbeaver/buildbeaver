package documents

import (
	"fmt"
	"net/http"

	"github.com/buildbeaver/buildbeaver/common/gerror"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/dto"
)

// Build contains information and links relating to a build resource, but not the jobs and steps it contains.
type Build struct {
	baseResourceDocument

	ID        models.BuildID `json:"id"`
	CreatedAt models.Time    `json:"created_at"`
	UpdatedAt models.Time    `json:"updated_at"`
	DeletedAt *models.Time   `json:"deleted_at,omitempty"`
	ETag      models.ETag    `json:"etag" hash:"ignore"`

	// Name of the build.
	Name models.ResourceName `json:"name"`
	// RepoID of the repo being built.
	RepoID models.RepoID `json:"repo_id"`
	// CommitID that is being built.
	CommitID models.CommitID `json:"commit_id"`
	// LogDescriptorID points to the log for this build.
	LogDescriptorID models.LogDescriptorID `json:"log_descriptor_id"`
	// Ref is the git ref the build is for (e.g. branch or tag)
	Ref string `json:"ref"`
	// Status reflects where the build is in the queue.
	Status models.WorkflowStatus `json:"status"`
	// Timings records the times at which the build transitioned between statuses.
	Timings WorkflowTimings `json:"timings"`
	// Error is set if the build finished with an error (or nil if the build succeeded).
	Error *models.Error `json:"error"`
	// Opts that are applied to this build.
	Opts BuildOptions `json:"opts"`

	LogDescriptorURL  string `json:"log_descriptor_url"`
	ArtifactSearchURL string `json:"artifact_search_url"`
}

func MakeBuild(rctx routes.RequestContext, build *models.Build) *Build {
	return &Build{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeBuildLink(rctx, build.ID),
		},

		ID:        build.ID,
		CreatedAt: build.CreatedAt,
		UpdatedAt: build.UpdatedAt,
		DeletedAt: build.DeletedAt,
		ETag:      build.ETag,

		Name:            build.Name,
		RepoID:          build.RepoID,
		CommitID:        build.CommitID,
		LogDescriptorID: build.LogDescriptorID,
		Ref:             build.Ref,
		Status:          build.Status,
		Timings:         *MakeWorkflowTimings(&build.Timings),
		Error:           build.Error,
		Opts:            *MakeBuildOptions(&build.Opts),

		LogDescriptorURL:  routes.MakeLogLink(rctx, build.LogDescriptorID),
		ArtifactSearchURL: routes.MakeArtifactSearchLink(rctx, build.ID),
	}
}

func MakeBuilds(rctx routes.RequestContext, builds []*models.Build) []*Build {
	var docs []*Build
	for _, model := range builds {
		docs = append(docs, MakeBuild(rctx, model))
	}
	return docs
}

func (d *Build) GetID() models.ResourceID {
	return d.ID.ResourceID
}

func (d *Build) GetKind() models.ResourceKind {
	return models.BuildResourceKind
}

func (d *Build) GetCreatedAt() models.Time {
	return d.CreatedAt
}

// BuildOptions contains options that affect how the build is scheduled or executed.
type BuildOptions struct {
	// Force all jobs in the build to run by ignoring fingerprints.
	Force bool `json:"force"`
	// NodesToRun contains zero or more workflows, jobs and steps to run. If no nodes are specified
	// then all workflows, jobs and steps will be run.
	NodesToRun []NodeFQN `json:"nodes_to_run"`
}

func MakeBuildOptions(opts *models.BuildOptions) *BuildOptions {
	return &BuildOptions{
		Force:      opts.Force,
		NodesToRun: MakeNodeFQNs(opts.NodesToRun),
	}
}

// NodeFQN is the Fully Qualified Name identifying a node in the build graph.
type NodeFQN struct {
	WorkflowName models.ResourceName `json:"workflow_name"`
	JobName      models.ResourceName `json:"job_name"`
	StepName     models.ResourceName `json:"step_name"`
}

func MakeNodeFQN(fqn models.NodeFQN) NodeFQN {
	return NodeFQN{
		WorkflowName: fqn.WorkflowName,
		JobName:      fqn.JobName,
		StepName:     fqn.StepName,
	}
}

func MakeNodeFQNs(fqns []models.NodeFQN) []NodeFQN {
	var docs []NodeFQN
	for _, fqn := range fqns {
		docs = append(docs, MakeNodeFQN(fqn))
	}
	return docs
}

func (s NodeFQN) String() string {
	if s.JobName == "" && s.StepName == "" {
		return s.WorkflowName.String()
	} else if s.StepName == "" {
		return fmt.Sprintf("%s.%s", s.WorkflowName, s.JobName)
	} else {
		return fmt.Sprintf("%s.%s.%s", s.WorkflowName, s.JobName, s.StepName)
	}
}

// BuildGraph contains information and links relating to a build resource, including all the jobs and steps
// currently queued for the build, as well as other details including the repo and commit for the build.
type BuildGraph struct {
	baseResourceDocument
	Build *Build `json:"build"`
	// Jobs that make up the build.
	Jobs []*JobGraph `json:"jobs"`
	// Repo that was committed to.
	Repo *Repo `json:"repo"`
	// Commit that the build was generated from.
	Commit *Commit `json:"commit"`
}

func MakeBuildGraph(rctx routes.RequestContext, build *dto.QueuedBuild) *BuildGraph {
	return &BuildGraph{
		baseResourceDocument: baseResourceDocument{
			URL: routes.MakeBuildLink(rctx, build.ID),
		},
		Repo:   MakeRepo(rctx, build.Repo),
		Commit: MakeCommit(rctx, build.Commit),
		Jobs:   MakeJobGraphs(rctx, build.Jobs),
		Build:  MakeBuild(rctx, build.BuildGraph.Build),
	}
}

func (d *BuildGraph) GetID() models.ResourceID {
	return d.Build.GetID()
}

func (d *BuildGraph) GetKind() models.ResourceKind {
	return d.Build.GetKind()
}

func (d *BuildGraph) GetCreatedAt() models.Time {
	return d.Build.GetCreatedAt()
}

type CreateBuildRequest struct {
	// FromBuildID nominates a previous build to clone to create the new build.
	// In the future, we may support specifying a commit/ref instead of a previous
	// build but for now this gives us "re-run" functionality.
	FromBuildID *models.BuildID      `json:"from_build_id"`
	Opts        *models.BuildOptions `json:"opts"`
}

func (d *CreateBuildRequest) Bind(r *http.Request) error {
	if d.FromBuildID == nil || !d.FromBuildID.Valid() {
		return gerror.NewErrValidationFailed("The build to base the new build on must be set")
	}
	return nil
}

// BuildSearchResult is the API layer representation of a BuildSearchResult that can be sent to the UI
type BuildSearchResult struct {
	// Build resource containing details of the build
	Build *Build `json:"build"`
	// Repo that was committed to.
	Repo *Repo `json:"repo"`
	// Commit that the build was generated from.
	Commit *Commit `json:"commit"`
}

func MakeBuildSearchResult(rctx routes.RequestContext, build *models.BuildSearchResult) *BuildSearchResult {
	return &BuildSearchResult{
		Repo:   MakeRepo(rctx, build.Repo),
		Commit: MakeCommit(rctx, build.Commit),
		Build:  MakeBuild(rctx, build.Build),
	}
}

func MakeBuildSearchResultsDocument(rctx routes.RequestContext, builds []*models.BuildSearchResult) []*BuildSearchResult {
	var queuedBuildsDocument []*BuildSearchResult
	for _, build := range builds {
		queuedBuildsDocument = append(queuedBuildsDocument, MakeBuildSearchResult(rctx, build))
	}
	return queuedBuildsDocument
}

type BuildSummary struct {
	Running   []*BuildSearchResult `json:"running"`
	Upcoming  []*BuildSearchResult `json:"upcoming"`
	Completed []*BuildSearchResult `json:"completed"`
}

func MakeBuildSummary(rctx routes.RequestContext, buildSummary *models.BuildSummaryResult) *BuildSummary {
	return &BuildSummary{
		Running:   MakeBuildSearchResultsDocument(rctx, buildSummary.Running),
		Upcoming:  MakeBuildSearchResultsDocument(rctx, buildSummary.Upcoming),
		Completed: MakeBuildSearchResultsDocument(rctx, buildSummary.Completed),
	}
}
