import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { SearchBar } from './search-bar.component';
import userEvent from '@testing-library/user-event';
import * as searchHook from '../../hooks/search/search.hook';
import { IUseSearch } from '../../hooks/search/search.hook';
import { BrowserRouter } from 'react-router-dom';
import { mockBuildGraph } from '../../mocks/models/build-graph.mock';
import { mockRepo } from '../../mocks/models/repo.mock';
import { mockUseLegalEntityById } from '../../mocks/hooks/useLegalEntityById.mock';
import { IStructuredError } from '../../interfaces/structured-error.interface';

// So we don't actually wait for the debounce in tests
jest.useFakeTimers();
mockUseLegalEntityById();

interface RenderOptions {
  search?: IUseSearch;
}

const defaultRenderOptions: RenderOptions = {
  search: {
    builds: undefined,
    isSearching: false,
    repos: undefined,
    searchError: undefined
  }
};

describe('SearchBar', () => {
  const renderSearchBar = (renderOptions?: RenderOptions): void => {
    const { search } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    jest.spyOn(searchHook, 'useSearch').mockImplementation(() => search as IUseSearch);

    render(
      <BrowserRouter>
        <SearchBar />
      </BrowserRouter>
    );
  };

  it('should render search results', async () => {
    const search: IUseSearch = {
      builds: [mockBuildGraph()],
      isSearching: false,
      repos: [mockRepo()],
      searchError: undefined
    };

    renderSearchBar({ search });

    const searchInput = screen.getByRole('textbox');
    userEvent.type(searchInput, 'test');

    // Builds section with a single build
    expect(await screen.findByText('Builds')).toBeInTheDocument();
    expect(
      screen.getByText((content, node: Element) => node.textContent === 'test-org / billys-playground #4')
    ).toBeInTheDocument();
    expect(screen.getByText('This is a test commit')).toBeInTheDocument();

    // Repos section with a single repo
    expect(screen.getByText('Repos')).toBeInTheDocument();
    expect(screen.getByText('Test repo')).toBeInTheDocument();
    expect(screen.getByText('A repo for testing')).toBeInTheDocument();
  });

  it('should render no matching results when results are empty', async () => {
    renderSearchBar();

    const searchInput = screen.getByRole('textbox');
    userEvent.type(searchInput, 'test');

    expect(await screen.findByText('No matching results')).toBeInTheDocument();
  });

  it('should be keyboard navigable', () => {
    const search: IUseSearch = {
      builds: [mockBuildGraph()],
      isSearching: false,
      repos: [mockRepo()],
      searchError: undefined
    };

    renderSearchBar({ search });

    expect(screen.queryByText('Builds')).toBeNull();

    const searchInput = screen.getByRole('textbox');

    // Focus the search input with forward slash key
    fireEvent.keyDown(document, { key: '/' });
    expect(searchInput).toHaveFocus();

    // Escape key should blur the input
    fireEvent.keyDown(document, { key: 'Escape' });
    expect(searchInput).not.toHaveFocus();
  });

  it('should show an error message when searching fails', async () => {
    const search: IUseSearch = {
      builds: undefined,
      isSearching: false,
      repos: undefined,
      searchError: {} as IStructuredError
    };

    renderSearchBar({ search });

    const searchInput = screen.getByRole('textbox');
    userEvent.type(searchInput, 'test');

    expect(await screen.findByText('Search failed')).toBeInTheDocument();
    expect(screen.queryByText('Builds')).toBeNull();
    expect(screen.queryByText('Repos')).toBeNull();
  });
});
