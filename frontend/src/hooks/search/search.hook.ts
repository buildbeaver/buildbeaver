import { useEffect, useState } from 'react';
import { search } from '../../services/root.service';
import { CancellableSearch } from '../../models/cancellable-search.model';
import { IResourceResponse } from '../../services/responses/resource-response.interface';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IRepo } from '../../interfaces/repo.interface';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IStructuredError } from '../../interfaces/structured-error.interface';

export interface IUseSearch {
  builds?: IBuildGraph[];
  isSearching: boolean;
  repos?: IRepo[];
  searchError?: IStructuredError;
}

type Resource = IBuildGraph | IRepo;

/**
 * Wrapper around universal search.
 */
export function useSearch(cancellableSearch: CancellableSearch): IUseSearch {
  const [isSearching, setIsSearching] = useState(false);
  const [builds, setBuilds] = useState<IBuildGraph[] | undefined>();
  const [repos, setRepos] = useState<IRepo[] | undefined>();
  const [searchError, setSearchError] = useState<IStructuredError | undefined>();

  const getResultsByKind = <T = Resource>(searchResults: IResourceResponse<T>[], kind: ResourceKind): T[] | undefined => {
    return searchResults?.find((searchResult) => searchResult.kind === kind)?.results;
  };

  useEffect(() => {
    const runSearch = async (): Promise<void> => {
      setIsSearching(true);

      await search(cancellableSearch.query)
        .then((response) => {
          // Ignore responses for cancelled searches. The user has continued typing making these results stale.
          if (!cancellableSearch.isCancelled) {
            setBuilds(getResultsByKind<IBuildGraph>(response.results, ResourceKind.Build));
            setRepos(getResultsByKind<IRepo>(response.results, ResourceKind.Repo));
          }
        })
        .catch((error: IStructuredError) => {
          setSearchError(error);
        })
        .finally(() => {
          setIsSearching(false);
        });
    };

    // Don't search if the user hasn't typed anything yet
    if (cancellableSearch.query.length > 0) {
      runSearch();
    }
  }, [cancellableSearch]);

  return {
    builds,
    isSearching,
    repos,
    searchError
  };
}
