import React from 'react';
import { render, waitFor } from '@testing-library/react';
import { RootDocumentProvider } from './root-document.provider';
import * as rootService from '../../services/root.service';
import { screen } from '@testing-library/dom';
import '@testing-library/jest-dom';

describe('Root document provider', () => {
  it('should render props.children after fetching the root document', async () => {
    const mock = jest.spyOn(rootService, 'fetchRootDocument');
    mock.mockResolvedValue({
      current_legal_entity_url: 'foo',
      github_authentication_url: 'bar',
      legal_entities_url: 'baz'
    });

    render(
      <RootDocumentProvider>
        <div>Children</div>
      </RootDocumentProvider>
    );

    expect(screen.getByTestId('loading')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('Children')).toBeInTheDocument();
    });

    expect(screen.queryByTestId('loading')).not.toBeInTheDocument();
  });

  it('should render an error after failing to fetch the root document', async () => {
    const mock = jest.spyOn(rootService, 'fetchRootDocument');
    mock.mockRejectedValue('Failed to fetch root document');

    render(
      <RootDocumentProvider>
        <div>Children</div>
      </RootDocumentProvider>
    );

    expect(screen.getByTestId('loading')).toBeInTheDocument();

    await waitFor(() => {
      expect(screen.getByText('BuildBeaver is unavailable. Please try again later.')).toBeInTheDocument();
    });

    expect(screen.queryByText('Children')).not.toBeInTheDocument();
    expect(screen.queryByTestId('loading')).not.toBeInTheDocument();
  });
});
