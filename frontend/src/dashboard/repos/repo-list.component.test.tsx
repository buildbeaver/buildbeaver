import React from 'react';
import { render, screen } from '@testing-library/react';
import { RepoList } from './repo-list.component';
import { mockUseStaticResourceList } from '../../mocks/hooks/useStaticResourceList.mock';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { mockOrg } from '../../mocks/models/org.mock';
import { BrowserRouter } from 'react-router-dom';
import { mockRepo } from '../../mocks/models/repo.mock';
import { IRepo } from '../../interfaces/repo.interface';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IStructuredError } from '../../interfaces/structured-error.interface';

describe('RepoList', () => {
  const renderRepoList = (): void => {
    render(
      <BrowserRouter>
        <CurrentLegalEntityContext.Provider value={{ currentLegalEntity: mockOrg() }}>
          <RepoList enabled={true} repoSearchUrl="" repoUpdated={() => {}} />
        </CurrentLegalEntityContext.Provider>
      </BrowserRouter>
    );
  };

  it('should render a message when there are no repos', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo);
    renderRepoList();

    expect(screen.getByText('No repos to display')).toBeInTheDocument();
  });

  it('should render a message when an error occurs loading repos', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo, { error: { message: 'Loading failed' } as IStructuredError });
    renderRepoList();

    expect(screen.getByText('Failed to load repos')).toBeInTheDocument();
  });

  it('should render repos', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo, { response: { next_url: '', prev_url: '', results: [mockRepo()] } });
    renderRepoList();

    expect(screen.getByText('Test repo')).toBeInTheDocument();
    expect(screen.getByText('A repo for testing')).toBeInTheDocument();
  });
});
