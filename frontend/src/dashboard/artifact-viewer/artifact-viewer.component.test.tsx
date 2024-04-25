import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { mockUseStaticResourceList } from '../../mocks/hooks/useStaticResourceList.mock';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ArtifactViewer } from './artifact-viewer.component';
import { IArtifactDefinition } from '../../interfaces/artifact-definition.interface';
import { mockJob } from '../../mocks/models/job.mock';
import { IJob } from '../../interfaces/job.interface';
import { Status } from '../../enums/status.enum';
import { mockArtifact } from '../../mocks/models/artifact.mock';

interface RenderOptions {
  job: IJob;
}

const defaultRenderOptions: RenderOptions = {
  job: mockJob()
};

describe('ArtifactViewer', () => {
  const renderArtifactViewer = (renderOptions?: RenderOptions): void => {
    const { job } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <ArtifactViewer artifactSearchUri="/foo" selectedJob={job} />
      </BrowserRouter>
    );
  };

  it('should render a message when an error occurs while fetching artifacts', () => {
    mockUseStaticResourceList<IArtifactDefinition>(ResourceKind.Artifact, { error: {} as IStructuredError });
    renderArtifactViewer();

    expect(screen.getByText('Failed to load artifacts')).toBeInTheDocument();
  });

  it('should render a message when the selected job has not finished', () => {
    const job = mockJob({ status: Status.Running });

    mockUseStaticResourceList<IArtifactDefinition>(ResourceKind.Artifact);
    renderArtifactViewer({ job });

    expect(screen.getByText('Artifacts will become available after the job has completed...')).toBeInTheDocument();
  });

  it('should render a message when there are no artifacts for a job', () => {
    mockUseStaticResourceList<IArtifactDefinition>(ResourceKind.Artifact, {
      response: {
        kind: ResourceKind.Artifact,
        next_url: '',
        prev_url: '',
        results: []
      }
    });
    renderArtifactViewer();

    expect(screen.getByText('No artifacts defined for this job')).toBeInTheDocument();
  });

  it('should render an artifacts table', () => {
    mockUseStaticResourceList<IArtifactDefinition>(ResourceKind.Artifact, {
      response: {
        kind: ResourceKind.Artifact,
        next_url: '',
        prev_url: '',
        results: [mockArtifact()]
      }
    });
    renderArtifactViewer();

    expect(screen.getByRole('columnheader', { name: 'Name' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'test-group' })).toBeInTheDocument();

    expect(screen.getByRole('columnheader', { name: 'Path' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'foo/bar/test-artifact.go' })).toBeInTheDocument();

    expect(screen.getByRole('columnheader', { name: 'Size' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: '13.1 kB' })).toBeInTheDocument();

    expect(screen.getByRole('columnheader', { name: '' })).toBeInTheDocument();
    expect(screen.getByRole('cell', { name: 'Download artifact' })).toBeInTheDocument();
  });

  describe('when a urls are included in the resource response', () => {
    it('should render pagination controls', () => {
      mockUseStaticResourceList<IArtifactDefinition>(ResourceKind.Artifact, {
        response: {
          kind: ResourceKind.Artifact,
          next_url: 'page-1',
          prev_url: 'page-3',
          results: [mockArtifact()]
        }
      });
      renderArtifactViewer();

      expect(screen.getByRole('button', { name: 'Prev' })).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument();
    });
  });
});
