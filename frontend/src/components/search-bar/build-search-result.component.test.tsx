import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';
import * as legalEntityByIdHook from '../../hooks/legal-entity-by-id/legal-entity-by-id.hook';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { BuildSearchResult } from './build-search-result.component';
import { mockBuildGraph } from '../../mocks/models/build-graph.mock';
import { mockUseLegalEntityById } from '../../mocks/hooks/useLegalEntityById.mock';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface RenderOptions {
  bGraph: IBuildGraph;
}

const defaultRenderOptions: RenderOptions = {
  bGraph: mockBuildGraph()
};

describe('BuildSearchResult', () => {
  const renderBuildSearchItem = (renderOptions?: RenderOptions): void => {
    const { bGraph } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <BuildSearchResult bGraph={bGraph} isFocused={false} isLast={false} clicked={() => {}} />
      </BrowserRouter>
    );
  };

  it('should render build details', () => {
    mockUseLegalEntityById();
    renderBuildSearchItem();

    expect(
      screen.getByText((content, node: Element) => node.textContent === 'test-org / billys-playground #4')
    ).toBeInTheDocument();
    expect(screen.getByText('This is a test commit')).toBeInTheDocument();
    expect(screen.getByText((content, node: Element) => node.textContent === 'Committed by Billy')).toBeInTheDocument();
    expect(screen.getByText((content, node: Element) => node.textContent === 'main / 6bdb713f0792')).toBeInTheDocument();
  });

  it('should render a content loader while fetching an un-cached legal entity', () => {
    jest.spyOn(legalEntityByIdHook, 'useLegalEntityById').mockImplementation(() => {
      return { legalEntityError: undefined };
    });
    renderBuildSearchItem();

    expect(screen.getByTitle('Loading...')).toBeInTheDocument();
    expect(screen.queryByText((content, node: Element) => node.textContent === 'test-org / test-repo #test-build')).toBeNull();
    expect(screen.queryByText('Test commit message')).toBeNull();
  });

  it('should render an error message when fetching legal entity by id fails', () => {
    jest.spyOn(legalEntityByIdHook, 'useLegalEntityById').mockImplementation(() => {
      return { legalEntityError: {} as IStructuredError };
    });
    renderBuildSearchItem();

    expect(screen.getByText('Failed to load search result')).toBeInTheDocument();
  });
});
