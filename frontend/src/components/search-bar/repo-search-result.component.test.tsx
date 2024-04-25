import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { IRepo } from '../../interfaces/repo.interface';
import { render, screen } from '@testing-library/react';
import * as legalEntityByIdHook from '../../hooks/legal-entity-by-id/legal-entity-by-id.hook';
import { RepoSearchResult } from './repo-search-result.component';
import { mockUseLegalEntityById } from '../../mocks/hooks/useLegalEntityById.mock';
import { mockRepo } from '../../mocks/models/repo.mock';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface RenderOptions {
  repo: IRepo;
}

const defaultRenderOptions: RenderOptions = {
  repo: mockRepo()
};

describe('RepoSearchResult', () => {
  const renderRepoSearchItem = (renderOptions?: RenderOptions): void => {
    const { repo } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <RepoSearchResult isFocused={false} isLast={false} repo={repo} clicked={() => {}} />
      </BrowserRouter>
    );
  };

  it('should render repo details', () => {
    mockUseLegalEntityById();
    renderRepoSearchItem();

    expect(screen.getByText('Test repo')).toBeInTheDocument();
    expect(screen.getByText('A repo for testing')).toBeInTheDocument();
  });

  it('should render a placeholder message for a repo with no description', () => {
    mockUseLegalEntityById();
    renderRepoSearchItem({ repo: { description: '', name: 'Test repo' } as IRepo });

    expect(screen.getByText('Test repo')).toBeInTheDocument();
    expect(screen.getByText('No description available')).toBeInTheDocument();
    expect(screen.queryByText('A repo for testing')).toBeNull();
  });

  it('should render a content loader while fetching an un-cached legal entity', () => {
    jest.spyOn(legalEntityByIdHook, 'useLegalEntityById').mockImplementation(() => {
      return { legalEntityError: undefined };
    });
    renderRepoSearchItem();

    expect(screen.getByTitle('Loading...')).toBeInTheDocument();

    expect(screen.queryByText('Test repo')).toBeNull();
    expect(screen.queryByText('A repo for testing')).toBeNull();
  });

  it('should render an error message when fetching legal entity by id fails', () => {
    jest.spyOn(legalEntityByIdHook, 'useLegalEntityById').mockImplementation(() => {
      return { legalEntityError: {} as IStructuredError };
    });
    renderRepoSearchItem();

    expect(screen.getByText('Failed to load search result')).toBeInTheDocument();
  });
});
