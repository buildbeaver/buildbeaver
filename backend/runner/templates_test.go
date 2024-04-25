package runner

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/server/api/rest/documents"
)

func TestTemplateJob(t *testing.T) {

	dep1 := &documents.Job{
		Name:        "test-dep1",
		Fingerprint: "123",
	}

	dep2 := &documents.Job{
		Name:        "test-dep2",
		Fingerprint: "abc",
	}

	job := &documents.RunnableJob{
		Job: &documents.Job{
			Name: "test-main",
			Depends: []*documents.JobDependency{
				{JobName: "test-dep1"},
				{JobName: "test-dep2"},
			},
			DockerConfig: &documents.DockerConfig{},
		},
		Jobs: []*documents.Job{dep1, dep2},
	}

	// Single match
	job.Job.DockerConfig.Image = "~^$yo ${{ jobs.test-dep1.fingerprint }}"
	err := templateJob(job)
	require.Nil(t, err)
	require.Equal(t, job.Job.DockerConfig.Image, "~^$yo 123")

	// Multiple matches
	job.Job.DockerConfig.Image = "~^$yo ${{ jobs.test-dep1.fingerprint }} this*ab54is12!&*(*random~^&T#^stuff&*(!''\" ${{ jobs.test-dep2.fingerprint}} *%#!whatsup"
	err = templateJob(job)
	require.Nil(t, err)
	require.Equal(t, job.Job.DockerConfig.Image, "~^$yo 123 this*ab54is12!&*(*random~^&T#^stuff&*(!''\" abc *%#!whatsup")

	// Invalid variables
	job.Job.DockerConfig.Image = "${{ jobs.test-dep.idontexist }}"
	err = templateJob(job)
	require.Error(t, err)
	job.Job.DockerConfig.Image = "${{ random.test-dep.idontexist }}"
	err = templateJob(job)
	require.Error(t, err)

	// Broken template syntax
	job.Job.DockerConfig.Image = "${{ jobs.test/dep.fingerprint }}"
	err = templateJob(job)
	require.Error(t, err)
}
