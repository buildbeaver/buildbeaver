import React, { useEffect, useState } from 'react';
import { useStaticResourceList } from '../../hooks/resources/resource-list.hook';
import { IRepo } from '../../interfaces/repo.interface';
import { RepoListItem } from './repo-list-item.component';
import { List } from '../../components/list/list.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { PlaceholderMessage } from '../../components/placeholder-message/placeholder-message.component';
import { Pagination } from '../../components/pagination/pagination.component';
import { UpdateToken } from '../../models/update-token.model';

interface Props {
  enabled: boolean;
  lastRepoUpdated?: string;
  repoSearchUrl: string;
  repoUpdated: (repoName: string, enabled: boolean) => void;
}

export function RepoList(props: Props): JSX.Element {
  const { enabled, lastRepoUpdated, repoSearchUrl, repoUpdated } = props;
  const [reposUrl, setReposUrl] = useState(repoSearchUrl);
  const [repoUpdateTokens, setRepoUpdateTokens] = useState<UpdateToken[]>([]);
  const { loading, error, response, refresh } = useStaticResourceList<IRepo>({
    url: reposUrl,
    query: {
      filters: [
        {
          field: 'enabled',
          operator: '=',
          value: `${enabled}`
        }
      ]
    }
  });

  useEffect(() => {
    setRepoUpdateTokens(repoUpdateTokens.filter((token) => token.isUpdating));
    refresh();
  }, [lastRepoUpdated]);

  const pageChanged = (url: string): void => {
    setReposUrl(url);
  };

  /**
   * Maintains a collection of cancellation tokens that lets us identify which list items should be showing a loading
   * spinner. This information persists even as the list items move around, as other list items complete their updates
   * and shift to the other repo list.
   */
  const registerUpdateToken = (token: UpdateToken): void => {
    setRepoUpdateTokens([...repoUpdateTokens, token]);
  };

  if (loading) {
    return <SimpleContentLoader numberOfRows={3} rowHeight={40} />;
  }

  if (error || !response) {
    return <PlaceholderMessage message="Failed to load repos" />;
  }

  if (response.results.length === 0) {
    return <PlaceholderMessage message="No repos to display" />;
  }

  return (
    <>
      <List>
        {response.results.map((repo: IRepo) => (
          <RepoListItem
            key={repo.id}
            isLoading={repoUpdateTokens.some((token) => token.isUpdating && token.id === repo.id)}
            repo={repo}
            repoUpdated={() => repoUpdated(repo.name, !enabled)}
            registerUpdateToken={registerUpdateToken}
          />
        ))}
      </List>
      <Pagination response={response} pageChanged={pageChanged} />
    </>
  );
}
