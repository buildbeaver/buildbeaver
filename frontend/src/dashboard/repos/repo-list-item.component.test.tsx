import React from 'react';
import { render, screen } from '@testing-library/react';
import { mockRepo } from '../../mocks/models/repo.mock';
import { SetupContext } from '../../contexts/setup/setup.context';
import { RepoListItem } from './repo-list-item.component';
import { BrowserRouter } from 'react-router-dom';

interface RenderOptions {
  isInSetupContext: boolean;
}

const defaultRenderOptions: RenderOptions = {
  isInSetupContext: false
};

describe('RepoListItem', () => {
  const renderRepoListItem = (renderOptions?: RenderOptions): void => {
    const { isInSetupContext } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <SetupContext.Provider value={{ isInSetupContext, setupPath: '', setupUrl: '' }}>
          <RepoListItem isLoading={false} repo={mockRepo()} registerUpdateToken={jest.fn()} repoUpdated={jest.fn()} />
        </SetupContext.Provider>
      </BrowserRouter>
    );
  };

  it('should render a clickable repo list item', () => {
    renderRepoListItem();

    expect(screen.getByRole('button', { name: 'Enable' })).toBeInTheDocument();
    expect(screen.getByRole('link', { name: 'Test repo A repo for testing Enable' })).toBeInTheDocument();
  });

  it('should render a non-clickable repo list item while in the context of legal entity setup', () => {
    renderRepoListItem({ isInSetupContext: true });

    expect(screen.getByRole('button', { name: 'Enable' })).toBeInTheDocument();
    expect(screen.queryByRole('link', { name: 'Test repo A repo for testing Enable' })).toBeNull();
  });
});
