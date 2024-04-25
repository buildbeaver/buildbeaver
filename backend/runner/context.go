package runner

import (
	"context"

	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

type JobBuildContext struct {
	ctx         context.Context
	job         *documents.RunnableJob
	logPipeline logging.LogPipeline
}

func NewJobBuildContext(ctx context.Context, job *documents.RunnableJob) *JobBuildContext {
	return &JobBuildContext{
		ctx:         ctx,
		job:         job,
		logPipeline: logging.NewNoOpLogPipeline(), // logPipeline should never be nil
	}
}

func (c *JobBuildContext) Ctx() context.Context {
	return c.ctx
}

func (c *JobBuildContext) Job() *documents.RunnableJob {
	return c.job
}

func (c *JobBuildContext) SetJob(job *documents.RunnableJob) {
	c.job = job
}

func (c *JobBuildContext) SetJobDocument(job *documents.Job) {
	c.job.Job = job
}

// SetLogPipeline associates a log pipeline with the context that can be written to, to add lines to the job's logs.
// This pipeline will be closed when the job context is closed.
// Any previously set log pipeline will be overwritten, without being flushed and closed.
func (c *JobBuildContext) SetLogPipeline(pipeline logging.LogPipeline) {
	c.logPipeline = pipeline
}

// ClearLogPipeline 'clears' any previous set log pipeline by setting it back to a no-op log pipeline.
// Any previously set log pipeline will be overwritten, without being flushed and closed.
func (c *JobBuildContext) ClearLogPipeline() {
	c.SetLogPipeline(logging.NewNoOpLogPipeline())
}

// LogPipeline returns the job's log pipeline. This function will never return nil; if SetLogPipeline() has not yet
// been called then a NoOpLogPipeline will be returned.
func (c *JobBuildContext) LogPipeline() logging.LogPipeline {
	return c.logPipeline
}

// IsJobIndirected gets the indirect status for the job. If true this job should not execute but should be marked as successful.
func (c *JobBuildContext) IsJobIndirected() bool {
	return c.job.Job.IndirectToJobID.Valid()
}

type StepBuildContext struct {
	step        *documents.Step
	logPipeline logging.LogPipeline
	*JobBuildContext
}

func NewStepBuildContext(base *JobBuildContext, step *documents.Step) *StepBuildContext {
	return &StepBuildContext{
		step:            step,
		JobBuildContext: base,
		logPipeline:     logging.NewNoOpLogPipeline(), // logPipeline should never be nil
	}
}

func (c *StepBuildContext) Step() *documents.Step {
	return c.step
}

func (c *StepBuildContext) SetStep(step *documents.Step) {
	c.step = step
}

// SetLogPipeline associates a log pipeline with the context that can be written to, to add lines to the step's logs.
// This pipeline will be closed when the job context is closed.
// Any previously set log pipeline will be overwritten, without being flushed and closed.
func (c *StepBuildContext) SetLogPipeline(pipeline logging.LogPipeline) {
	c.logPipeline = pipeline
}

// ClearLogPipeline 'clears' any previous set log pipeline by setting it back to a no-op log pipeline.
// Any previously set log pipeline will be overwritten, without being flushed and closed.
func (c *StepBuildContext) ClearLogPipeline() {
	c.SetLogPipeline(logging.NewNoOpLogPipeline())
}

// LogPipeline returns the step's log pipeline. This function will never return nil; if SetLogPipeline() has not yet
// been called then a NoOpLogPipeline will be returned.
func (c *StepBuildContext) LogPipeline() logging.LogPipeline {
	return c.logPipeline
}
