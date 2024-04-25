import React, { useContext, useEffect, useState } from 'react';
import { IRepo } from '../../interfaces/repo.interface';
import { TickContext } from '../../contexts/tick/tick.context';
import { List } from '../../components/list/list.component';
import { ISecret } from '../../interfaces/secret.interface';
import { RepoSecretListItem } from './repo-secrets-list-item.component';
import { PlaceholderMessage } from '../../components/placeholder-message/placeholder-message.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { Pagination } from '../../components/pagination/pagination.component';
import { useStaticResourceList } from '../../hooks/resources/resource-list.hook';
import { StructuredError } from '../../components/structured-error/structured-error.component';

interface Props {
  repo: IRepo;
}

export function RepoSecretsList(props: Props): JSX.Element {
  const { repo } = props;
  const { tick } = useContext(TickContext);
  const [secretsUrl, setSecretsUrl] = useState(repo.secrets_url);
  const {
    loading,
    error,
    response: secretsResponse,
    refresh: refreshSecrets
  } = useStaticResourceList<ISecret>({ url: secretsUrl });

  useEffect(() => {
    refreshSecrets();
  }, [tick]);

  const pageChanged = (url: string): void => {
    setSecretsUrl(url);
  };

  if (loading) {
    return <SimpleContentLoader numberOfRows={3} rowHeight={40} />;
  }

  if (error) {
    return <StructuredError error={error} fallback="Failed to load secrets" />;
  }

  if (!secretsResponse || secretsResponse.results.length === 0) {
    return <PlaceholderMessage message="No secrets to display" />;
  }

  return (
    <>
      <List>
        {secretsResponse.results.map((secret: ISecret, index: number) => (
          <RepoSecretListItem key={index} secret={secret} />
        ))}
      </List>
      <Pagination response={secretsResponse} pageChanged={pageChanged} />
    </>
  );
}
