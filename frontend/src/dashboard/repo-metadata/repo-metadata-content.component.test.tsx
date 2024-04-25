import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';
import { mockBuildGraph } from '../../mocks/models/build-graph.mock';
import { RepoMetadataContent } from './repo-metadata-content.component';
import { mockRepo } from '../../mocks/models/repo.mock';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { mockUseLiveResourceList } from '../../mocks/hooks/useLiveResourceList.mock';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';

describe('RepoMetadataContent', () => {
  const renderRepoMetadataContent = (): void => {
    const repo = mockRepo();

    render(
      <BrowserRouter>
        <RepoMetadataContent repo={repo} />
      </BrowserRouter>
    );
  };

  describe('when a repo has at least one build', () => {
    it('should render repo metadata for the latest build', () => {
      mockUseLiveResourceList<IBuildGraph>(ResourceKind.Build, {
        response: { next_url: '', prev_url: '', results: [mockBuildGraph()] }
      });

      renderRepoMetadataContent();

      expect(screen.getByText('main')).toBeInTheDocument();
      expect(screen.getByText('Succeeded')).toBeInTheDocument();

      expect(screen.getByText('Builds')).toBeInTheDocument();
      expect(screen.getByText('4')).toBeInTheDocument();

      expect(screen.getByText('Description')).toBeInTheDocument();
      expect(screen.getByText('A repo for testing')).toBeInTheDocument();
    });
  });

  describe('when a repo has no builds', () => {
    it('should render repo metadata only', () => {
      mockUseLiveResourceList<IBuildGraph>(ResourceKind.Build, {
        response: { next_url: '', prev_url: '', results: [] }
      });

      renderRepoMetadataContent();

      expect(screen.getByText('main')).toBeInTheDocument();
      expect(screen.getByText('-----')).toBeInTheDocument();

      expect(screen.getByText('Builds')).toBeInTheDocument();
      expect(screen.getByText('0')).toBeInTheDocument();

      expect(screen.getByText('Description')).toBeInTheDocument();
      expect(screen.getByText('A repo for testing')).toBeInTheDocument();
    });
  });

  describe('when fetching the latest build fails', () => {
    it('should render repo metadata only', () => {
      mockUseLiveResourceList<IBuildGraph>(ResourceKind.Build, {
        error: {} as IStructuredError,
        loading: false,
        response: undefined
      });

      renderRepoMetadataContent();

      expect(screen.getByText('main')).toBeInTheDocument();
      expect(screen.getByText('Unknown')).toBeInTheDocument();

      expect(screen.getByText('Builds')).toBeInTheDocument();
      expect(screen.getByText('-----')).toBeInTheDocument();

      expect(screen.getByText('Description')).toBeInTheDocument();
      expect(screen.getByText('A repo for testing')).toBeInTheDocument();
    });
  });
});
