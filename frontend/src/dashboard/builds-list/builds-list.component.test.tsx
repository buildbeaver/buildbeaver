import React from 'react';
import { render, screen } from '@testing-library/react';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { BuildsList } from './builds-list.component';
import { BrowserRouter } from 'react-router-dom';
import { mockBuildGraph } from '../../mocks/models/build-graph.mock';
import { mockUseAnyLegalEntity } from '../../mocks/hooks/useAnyLegalEntity.mock';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface RenderOptions {
  builds?: IBuildGraph[];
  error?: IStructuredError;
}

const defaultRenderOptions: RenderOptions = {
  builds: [],
  error: undefined
};

describe('BuildsList', () => {
  const renderBuildsList = (renderOptions?: RenderOptions): void => {
    const { builds, error } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <BuildsList builds={builds} error={error} />
      </BrowserRouter>
    );
  };

  it('should render a message when no builds are provided', () => {
    renderBuildsList();

    expect(screen.getByText('No builds to display')).toBeInTheDocument();
  });

  it('should render a message when an error is provided', () => {
    const error = { message: 'Something went wrong' } as IStructuredError;

    renderBuildsList({ error });

    expect(screen.getByText('Failed to load builds')).toBeInTheDocument();
  });

  it('should render a message when an error and builds are provided', () => {
    const error = { message: 'Something went wrong' } as IStructuredError;
    const builds = [mockBuildGraph()];

    renderBuildsList({ builds, error });

    expect(screen.getByText('Failed to load builds')).toBeInTheDocument();
  });

  it('should render builds when no error is provided', () => {
    const builds = [mockBuildGraph()];

    mockUseAnyLegalEntity();
    renderBuildsList({ builds });

    expect(
      screen.getByText((content, node: Element) => node.textContent === 'test-org / billys-playground #4')
    ).toBeInTheDocument();
  });
});
