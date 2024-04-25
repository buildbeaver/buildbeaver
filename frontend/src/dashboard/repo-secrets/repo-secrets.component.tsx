import React from 'react';
import { useRepo } from '../../hooks/repo/repo.hook';
import { useRepoUrl } from '../../hooks/repo-url/repo-url.hook';
import { RepoSecretsList } from './repo-secrets-list.component';
import { NavLink } from 'react-router-dom';
import { Button } from '../../components/button/button.component';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { RepoMetadata } from '../repo-metadata/repo-metadata.component';
import { TickProvider } from '../../contexts/tick/tick.provider';

export function RepoSecrets(): JSX.Element {
  const repoUrl = useRepoUrl();
  const { repo, repoError } = useRepo(repoUrl);

  return (
    <>
      <TickProvider>
        <RepoMetadata repo={repo} repoError={repoError} />
      </TickProvider>
      <div className="my-6 flex">
        {repo && (
          <>
            <div className="flex grow flex-col">
              <div>
                Secrets are exported as environment variables during all builds that belong to this repo.<br></br> Existing secret
                values are hidden and cannot be viewed.
              </div>
            </div>
            <div className="flex justify-end">
              <NavLink to="new">
                <Button label="Create" />
              </NavLink>
            </div>
          </>
        )}
        {repoError && <StructuredError error={repoError} fallback="Failed to fetch repo information" handleNotFound={true} />}
      </div>
      <div className="my-6 flex flex-col gap-y-4">{repo && <RepoSecretsList repo={repo} />}</div>
    </>
  );
}
