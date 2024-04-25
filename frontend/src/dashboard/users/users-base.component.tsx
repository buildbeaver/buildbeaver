import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { RunnersBase } from '../runners/runners-base.component';
import { TickProvider } from '../../contexts/tick/tick.provider';
import { RepoBuildsBase } from '../repo-builds/repo-builds-base.component';
import { RepoSecretsBase } from '../repo-secrets/repo-secrets-base.component';
import { ReposBase } from '../repos/repos-base.component';
import { SetupWizardBase } from '../setup-wizard/setup-wizard-base.component';

export function UsersBase(): JSX.Element {
  return (
    <Routes>
      <Route
        path="repos"
        element={
          <TickProvider>
            <ReposBase />
          </TickProvider>
        }
      />
      <Route path="repos/:repo_name/builds/*" element={<RepoBuildsBase />} />
      <Route path="repos/:repo_name/secrets/*" element={<RepoSecretsBase />} />
      <Route path="repos/:repo_name" element={<Navigate to="builds" replace />} />
      <Route path="runners/*" element={<RunnersBase />} />
      <Route path="setup" element={<SetupWizardBase />} />
      {/*
        TODO: This is hidden until we have content to show at this route
        <Route path="settings" element={<LegalEntitySettings />} />
      */}
      <Route path="/" element={<Navigate to="repos" replace />} />
      <Route path="*" element={<Navigate to="/builds" replace />} />
    </Routes>
  );
}
