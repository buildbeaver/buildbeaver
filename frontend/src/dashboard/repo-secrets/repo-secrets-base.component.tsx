import React, { useContext, useMemo } from 'react';
import { Navigate, Route, Routes, useParams } from 'react-router-dom';
import { RepoSecrets } from './repo-secrets.component';
import { SecretCreate } from '../repo-secret/repo-secret-create.component';
import { ViewCentered } from '../../components/view-centered/view-centered.component';
import { RepoSecretBase } from '../repo-secret/repo-secret-base.component';
import { makeRepoNavigationTabs, RepoTab } from '../../utils/navigation-utils';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';

/**
 * Handles routing for views related to secrets in the context of a repo.
 */
export function RepoSecretsBase(): JSX.Element {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { repo_name } = useParams();
  const navigationTabs = useMemo(() => makeRepoNavigationTabs(RepoTab.Secrets, currentLegalEntity, repo_name!), []);

  return (
    <ViewCentered navigationTabs={navigationTabs}>
      <Routes>
        <Route path=":secret_id/*" element={<RepoSecretBase />} />
        <Route path="new" element={<SecretCreate />} />
        <Route path="/" element={<RepoSecrets />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ViewCentered>
  );
}
