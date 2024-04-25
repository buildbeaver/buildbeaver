package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/pkg/errors"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

// Dequeue returns the next build job that is ready to be executed, or nil if there are currently no queued builds.
func (a *APIClient) Dequeue(ctx context.Context) (*documents.RunnableJob, error) {
	url := "/api/v1/runner/queue"
	code, _, body, err := a.get(ctx, nil, url)

	if err != nil {
		return nil, err
	}
	if code == 404 {
		return nil, nil
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return nil, a.makeHTTPError(code, body)
	}
	doc := &documents.RunnableJob{}
	err = json.Unmarshal(body, doc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	// RunnableJob document must contain a job
	if doc.Job == nil {
		return nil, fmt.Errorf("error parsing response body: no 'job' element specified in RunnableJob")
	}

	return doc, nil
}

// Ping acts as a pre-flight check for a runner, contacting the server and checking that authentication
// and registration are in place ready to dequeue build jobs.
func (a *APIClient) Ping(ctx context.Context) error {
	url := "/api/v1/runner/ping"
	code, _, body, err := a.get(ctx, nil, url)

	if err != nil {
		return err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return a.makeHTTPError(code, body)
	}

	return nil
}

// SendRuntimeInfo sends information about the runtime environment and version for this runner to the server.
func (a *APIClient) SendRuntimeInfo(ctx context.Context, info *documents.PatchRuntimeInfoRequest) error {
	url := "/api/v1/runner/runtime"
	code, _, body, err := a.patch(ctx, nil, url, info)
	if err != nil {
		return err
	}
	if !a.isOneOf(code, []int{http.StatusOK}) {
		return a.makeHTTPError(code, body)
	}

	return nil
}

// UpdateJobStatus updates the status of the specified job.
// If the status is finished, err can be supplied to signal the job failed with an error
// or nil to signify the job succeeded.
func (a *APIClient) UpdateJobStatus(
	ctx context.Context,
	jobID models.JobID,
	status models.WorkflowStatus,
	jobError *models.Error,
	eTag models.ETag) (*documents.Job, error) {

	doc := &documents.PatchJobRequest{
		Status: &status,
		Error:  jobError,
	}
	url := fmt.Sprintf("/api/v1/runner/jobs/%s", jobID)
	code, _, body, err := a.patch(ctx, a.ifMatchHeader(eTag), url, doc)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK, http.StatusNoContent}) {
		return nil, a.makeHTTPError(code, body)
	}
	resDoc := &documents.Job{}
	err = json.Unmarshal(body, resDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return resDoc, nil
}

// UpdateJobFingerprint sets the fingerprint that has been calculated for a job. If the build is not configured
// with the force option (e.g. force=false), the server will attempt to locate a previously successful job with a
// matching fingerprint and indirect this job to it. If an indirection has been set, the agent must skip the job.
func (a *APIClient) UpdateJobFingerprint(
	ctx context.Context,
	jobID models.JobID,
	jobFingerprint string,
	jobFingerprintHashType *models.HashType,
	eTag models.ETag) (*documents.Job, error) {

	doc := &documents.PatchJobRequest{
		Fingerprint:         &jobFingerprint,
		FingerprintHashType: jobFingerprintHashType,
	}
	url := fmt.Sprintf("/api/v1/runner/jobs/%s", jobID)
	code, _, body, err := a.patch(ctx, a.ifMatchHeader(eTag), url, doc)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK, http.StatusNoContent}) {
		return nil, a.makeHTTPError(code, body)
	}
	resDoc := &documents.Job{}
	err = json.Unmarshal(body, resDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return resDoc, nil
}

// UpdateStepStatus updates the status of the specified step.
// If the status is finished, err can be supplied to signal the step failed with an error
// or nil to signify the step succeeded.
func (a *APIClient) UpdateStepStatus(
	ctx context.Context,
	stepID models.StepID,
	status models.WorkflowStatus,
	stepError *models.Error,
	eTag models.ETag) (*documents.Step, error) {

	doc := &documents.PatchStepRequest{
		Status: &status,
		Error:  stepError,
	}
	url := fmt.Sprintf("/api/v1/runner/steps/%s", stepID)
	code, _, body, err := a.patch(ctx, a.ifMatchHeader(eTag), url, doc)
	if err != nil {
		return nil, err
	}
	if !a.isOneOf(code, []int{http.StatusOK, http.StatusNoContent}) {
		return nil, a.makeHTTPError(code, body)
	}
	resDoc := &documents.Step{}
	err = json.Unmarshal(body, resDoc)
	if err != nil {
		return nil, errors.Wrapf(err, "error parsing response body: %s", string(body[:]))
	}
	return resDoc, nil
}
