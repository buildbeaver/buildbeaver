import React from 'react';
import { render, screen } from '@testing-library/react';
import { BuildsSidebar } from './builds-sidebar.component';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';
import { MemoryRouter } from 'react-router';
import { LegalEntitiesContext } from '../../contexts/legal-entities/legal-entities.context';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';
import { IRepo } from '../../interfaces/repo.interface';
import { mockUseStaticResourceList } from '../../mocks/hooks/useStaticResourceList.mock';
import { mockOrg } from '../../mocks/models/org.mock';
import { mockUser } from '../../mocks/models/user.mock';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { mockRepo } from '../../mocks/models/repo.mock';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface RenderOptions {
  selectedLegalEntity?: ILegalEntity;
}

const defaultRenderOptions: RenderOptions = {
  selectedLegalEntity: mockOrg()
};

describe('BuildsSidebar', () => {
  const renderBuildsSidebar = (renderOptions?: RenderOptions): void => {
    const { selectedLegalEntity } = {
      ...defaultRenderOptions,
      ...renderOptions
    };
    const legalEntitiesContextProviderValue = {
      legalEntities: [selectedLegalEntity]
    } as any;

    const selectedLegalEntityContextProviderValue = {
      selectedLegalEntity
    } as any;

    render(
      <MemoryRouter initialEntries={['/builds']}>
        <LegalEntitiesContext.Provider value={legalEntitiesContextProviderValue}>
          <SelectedLegalEntityContext.Provider value={selectedLegalEntityContextProviderValue}>
            <BuildsSidebar />
          </SelectedLegalEntityContext.Provider>
        </LegalEntitiesContext.Provider>
      </MemoryRouter>
    );
  };

  it('should render for an org', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo);
    renderBuildsSidebar();

    expect(screen.getByText('View Organization')).toBeInTheDocument();
    expect(screen.getByText('No repositories enabled')).toBeInTheDocument();
  });

  it('should render for a user', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo);
    renderBuildsSidebar({ ...defaultRenderOptions, selectedLegalEntity: mockUser() });

    expect(screen.getByText('Your Home')).toBeInTheDocument();
    expect(screen.getByText('No repositories enabled')).toBeInTheDocument();
  });

  it('should render enabled repos', () => {
    const mockEnabledRepos = [
      {
        id: 'repo-1',
        name: 'Repo 1'
      } as IRepo,
      {
        id: 'repo-2',
        name: 'Repo 2'
      } as IRepo
    ];

    mockUseStaticResourceList<IRepo>(ResourceKind.Repo, { response: { next_url: '', prev_url: '', results: mockEnabledRepos } });
    renderBuildsSidebar();

    expect(screen.getByText('Repo 1')).toBeInTheDocument();
    expect(screen.getByText('Repo 2')).toBeInTheDocument();
  });

  it('should render a message when an error occurs loading repos', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo, { error: { message: 'Loading failed' } as IStructuredError });
    renderBuildsSidebar();

    expect(screen.getByText('Failed to load repos')).toBeInTheDocument();
  });

  it('should render "View all" when there is more than one page of repo results', () => {
    mockUseStaticResourceList<IRepo>(ResourceKind.Repo, {
      response: { next_url: 'some-next-url', prev_url: '', results: [mockRepo()] }
    });
    renderBuildsSidebar();

    expect(screen.getByText('View all')).toBeInTheDocument();
  });
});
