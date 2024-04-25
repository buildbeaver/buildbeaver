import React from 'react';
import { render, screen } from '@testing-library/react';
import { RepoSecretsList } from './repo-secrets-list.component';
import { mockRepo } from '../../mocks/models/repo.mock';
import { mockUseStaticResourceList } from '../../mocks/hooks/useStaticResourceList.mock';
import { mockSecret } from '../../mocks/models/secret.mock';
import { BrowserRouter } from 'react-router-dom';
import { ISecret } from '../../interfaces/secret.interface';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IStructuredError } from '../../interfaces/structured-error.interface';

describe('RepoSecretsList', () => {
  const renderRepoSecretsList = (): void => {
    render(
      <BrowserRouter>
        <RepoSecretsList repo={mockRepo()} />
      </BrowserRouter>
    );
  };

  it('should render a message when there are no secrets', () => {
    mockUseStaticResourceList<ISecret>(ResourceKind.Secret);
    renderRepoSecretsList();

    expect(screen.getByText('No secrets to display')).toBeInTheDocument();
  });

  it('should render a message when an error occurs loading secrets', () => {
    mockUseStaticResourceList<ISecret>(ResourceKind.Secret, { error: { message: 'Loading failed' } as IStructuredError });
    renderRepoSecretsList();

    expect(screen.getByText('Failed to load secrets')).toBeInTheDocument();
  });

  it('should render secrets', () => {
    mockUseStaticResourceList<ISecret>(ResourceKind.Secret, {
      response: { next_url: '', prev_url: '', results: [mockSecret()] }
    });
    renderRepoSecretsList();

    expect(screen.getByText('Test secret')).toBeInTheDocument();
  });
});
