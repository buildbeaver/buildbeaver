import React, { useMemo, useRef, useState } from 'react';
import { IoCloseSharp, IoSearchSharp } from 'react-icons/io5';
import { CgFormatSlash } from 'react-icons/cg';
import { debounce } from 'lodash';
import { useSearch } from '../../hooks/search/search.hook';
import { Loading } from '../loading/loading.component';
import { CancellableSearch } from '../../models/cancellable-search.model';
import { useClickOutside } from '../../hooks/click-outside/click-outside.hook';
import { useKeyDownListener } from '../../hooks/key-down/key-down-listener.hook';
import './search-bar.component.scss';
import { isActiveElementTextInput } from '../../utils/document.utils';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IRepo } from '../../interfaces/repo.interface';
import { BuildSearchResult } from './build-search-result.component';
import { RepoSearchResult } from './repo-search-result.component';
import { StructuredError } from '../structured-error/structured-error.component';

export function SearchBar(): JSX.Element {
  const node = useRef<HTMLDivElement>(null);
  const searchInput = useRef<HTMLInputElement>(null);
  const [isFocused, setIsFocused] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [query, setQuery] = useState('');
  const [cancellableSearch, setCancellableSearch] = useState<CancellableSearch>(new CancellableSearch(''));
  const [focusedSearchResultIndex, setFocusedSearchResultIndex] = useState(-1);
  const { builds, isSearching, repos, searchError } = useSearch(cancellableSearch);

  const resultIds = [...(builds?.map((bGraph) => bGraph.build.id) ?? []), ...(repos?.map((repo) => repo.id) ?? [])];
  const focusedSearchResultId = resultIds[focusedSearchResultIndex];
  const hasQuery = query.length > 0;
  const showSearchResults = hasQuery && isFocused;
  const refs = new Map<string, React.RefObject<HTMLAnchorElement>>();

  const blurSearchInput = (): void => {
    closeSearchResults();
    searchInput.current?.blur();
  };

  /**
   * Cancels the previous search. If it was in progress its results won't return from the useSearch hook. Otherwise,
   * this is a no-op.
   */
  const cancelSearch = (): void => {
    cancellableSearch.cancel();
  };

  /**
   * Closes the search results area and clears the typed in query.
   */
  const clearAndCloseSearchResults = (): void => {
    setQuery('');
    closeSearchResults();
    searchInput.current?.focus();
  };

  /**
   * Don't remember the focused search result after closing the search bar
   */
  const clearFocusedSearchResult = (): void => {
    setFocusedSearchResultIndex(-1);
  };

  /**
   * Closes the search results area but retains the typed in query and search results.
   */
  const closeSearchResults = (): void => {
    cancelSearch();
    setIsLoading(false);
    setCancellableSearch(new CancellableSearch(''));
    setIsFocused(false);
    setFocusedSearchResultIndex(-1);
  };

  useClickOutside(
    node,
    () => {
      // Without this check, closeSearchResults() will fire on every click across the whole app
      if (isFocused) {
        closeSearchResults();
      }
    },
    [isFocused]
  );

  const focusSearchInput = (): void => {
    searchInput.current?.focus();
    setIsFocused(true);
  };

  /**
   * The last build will render with different styling if there are no repos.
   */
  const isLastBuild = (index: number, builds: IBuildGraph[], repos: IRepo[]): boolean => {
    return !repos && index === builds.length - 1;
  };

  /**
   * The last repo will always render with different styling.
   */
  const isLastRepo = (index: number, repos: IRepo[]): boolean => {
    return index === repos.length - 1;
  };

  /**
   * Key up and down through results using the up and down arrow keys. Rolls over to the top of the list when keying
   * down. Rolls over to the bottom of the list when keying up.
   * @param direction
   */
  const keyboardNavigateSearchResults = (direction: 'down' | 'up') => {
    if (!resultIds || resultIds.length === 0) {
      return;
    }

    const isDownDirection = direction === 'down';
    const indexModifier = isDownDirection ? 1 : -1;

    let nextIndex = focusedSearchResultIndex + indexModifier;

    const lowerBound = 0;
    const upperBound = resultIds.length - 1;
    const isOutOfBounds = nextIndex < lowerBound || nextIndex > upperBound;

    if (isDownDirection && isOutOfBounds) {
      nextIndex = lowerBound;
    } else if (isOutOfBounds) {
      nextIndex = upperBound;
    }

    setFocusedSearchResultIndex(nextIndex);

    const searchResultId = resultIds[nextIndex];
    const searchResultRef = refs.get(searchResultId);

    searchResultRef?.current?.scrollIntoView({ behavior: 'smooth', block: 'center', inline: 'nearest' });
  };

  /**
   * Programmatically clicks the simulated focused search result which triggers its internal <NavLink />
   */
  const navigateToFocusedSearchResult = (): void => {
    const searchResultId = resultIds[focusedSearchResultIndex];
    const searchResultRef = refs.get(searchResultId);

    blurSearchInput();
    searchResultRef?.current?.click();
    clearFocusedSearchResult();
  };

  /**
   * Tracks focus so that we can show persisted search results when:
   *  a) The user focuses the search input and
   *  b) The search input already has a query typed in.
   */
  const onFocus = (): void => {
    setIsFocused(true);
  };

  /**
   * Text input event handler. Triggers the search flow.
   * @param event - Contains the text typed by the user.
   */
  const queryChanged = (event: React.ChangeEvent<HTMLInputElement>): void => {
    const query = event.target.value;

    cancelSearch();
    setQuery(query);
    setIsLoading(true);
    clearFocusedSearchResult();
    searchWithDebounce(query);
  };

  /**
   * Searches with a debounced delay so that we don't fire a search query on every key stroke.
   */
  const searchWithDebounce = useMemo(
    () =>
      debounce((query: string): void => {
        // Sets a reference to this search so that we can cancel it if the user continues typing
        setCancellableSearch(new CancellableSearch(query));
        setIsLoading(false);
      }, 250),
    []
  );

  useKeyDownListener(
    '/',
    () => focusSearchInput(),
    () => isFocused || isActiveElementTextInput()
  );
  useKeyDownListener(
    'Escape',
    () => {
      blurSearchInput();
      clearFocusedSearchResult();
    },
    () => !isFocused
  );
  useKeyDownListener(
    'ArrowDown',
    () => keyboardNavigateSearchResults('down'),
    () => !isFocused
  );
  useKeyDownListener(
    'ArrowUp',
    () => keyboardNavigateSearchResults('up'),
    () => !isFocused
  );
  useKeyDownListener(
    'Enter',
    () => navigateToFocusedSearchResult(),
    () => !isFocused || !focusedSearchResultId
  );

  const renderBuildResults = (builds: IBuildGraph[], repos: IRepo[]): JSX.Element[] => {
    return builds.map((bGraph, index) => {
      const ref = React.createRef<HTMLAnchorElement>();

      refs.set(bGraph.build.id, ref);

      return (
        <BuildSearchResult
          bGraph={bGraph}
          isFocused={bGraph.build.id === focusedSearchResultId}
          isLast={isLastBuild(index, builds, repos)}
          key={index}
          clicked={closeSearchResults}
          ref={ref}
        />
      );
    });
  };

  const renderRepoResults = (repos: IRepo[]): JSX.Element[] => {
    return repos.map((repo, index) => {
      const ref = React.createRef<HTMLAnchorElement>();

      refs.set(repo.id, ref);

      return (
        <RepoSearchResult
          isFocused={repo.id === focusedSearchResultId}
          isLast={isLastRepo(index, repos)}
          key={index}
          repo={repo}
          clicked={closeSearchResults}
          ref={ref}
        />
      );
    });
  };

  const renderSearchResults = (): JSX.Element => {
    if (searchError) {
      return (
        <div className="flex h-[50px] w-full rounded-b-md">
          <StructuredError error={searchError} fallback="Search failed" handleNotFound={false} />
        </div>
      );
    }

    if (!repos && !builds) {
      return <div className="flex h-[50px] w-full items-center justify-center rounded-b-md bg-gray-100">No matching results</div>;
    }

    return (
      <div className="flex w-full flex-col">
        {builds && (
          <div className="flex flex-col">
            <p className="border-b bg-gray-100 p-1 text-center">Builds</p>
            {renderBuildResults(builds, repos ?? [])}
          </div>
        )}
        {repos && (
          <div className="flex w-full flex-col">
            <p className="border-b bg-gray-100 p-1 text-center">Repos</p>
            {renderRepoResults(repos)}
          </div>
        )}
      </div>
    );
  };

  return (
    <div className="search-bar relative flex h-[35px] w-full text-gray-600" ref={node}>
      <IoSearchSharp className="absolute left-3 h-[35px]" size={20} />
      <div className="absolute right-3 flex h-[35px] items-center">
        {hasQuery && (
          <div className="hover:text-gray-400">
            <IoCloseSharp className="cursor-pointer" size={24} onClick={clearAndCloseSearchResults} />
          </div>
        )}
        {!isFocused && (
          <div className="rounded bg-gray-200">
            <CgFormatSlash size={24} />
          </div>
        )}
      </div>
      <input
        className={`w-full border px-10 ${showSearchResults ? 'rounded-t-md' : 'rounded-md'}`}
        type="text"
        placeholder="Search BuildBeaver"
        onFocus={onFocus}
        ref={searchInput}
        value={query}
        onChange={queryChanged}
      />
      {showSearchResults && (
        <div className="absolute top-[39px] z-20 flex max-h-[300px] w-full overflow-y-auto rounded-b-md border border-paleSky bg-white text-sm text-gray-600 shadow-md">
          {isLoading || isSearching ? (
            <div className="flex h-[50px] w-full items-center justify-center rounded-b-md bg-gray-100">
              <Loading />
            </div>
          ) : (
            renderSearchResults()
          )}
        </div>
      )}
    </div>
  );
}
