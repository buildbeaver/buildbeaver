import React, { useContext, useMemo } from 'react';
import { ViewCentered } from '../../components/view-centered/view-centered.component';
import { useRepo } from '../../hooks/repo/repo.hook';
import { useRepoUrl } from '../../hooks/repo-url/repo-url.hook';
import { RepoBuildsList } from './repo-builds-list.component';
import { TickProvider } from '../../contexts/tick/tick.provider';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { RepoMetadata } from '../repo-metadata/repo-metadata.component';
import { makeRepoNavigationTabs, RepoTab } from '../../utils/navigation-utils';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { useParams } from 'react-router-dom';

export function RepoBuilds(): JSX.Element {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { repo_name } = useParams();
  const repoUrl = useRepoUrl();
  const { repo, repoError } = useRepo(repoUrl);
  const navigationTabs = useMemo(() => makeRepoNavigationTabs(RepoTab.Builds, currentLegalEntity, repo_name!), []);

  return (
    <ViewCentered navigationTabs={navigationTabs}>
      <RepoMetadata repo={repo} repoError={repoError} />
      <TickProvider>
        <div className="my-6">
          {repo && <RepoBuildsList repo={repo} />}
          {repoError && <StructuredError error={repoError} fallback="Failed to fetch repo information" handleNotFound={true} />}
        </div>
      </TickProvider>
    </ViewCentered>
  );
}
