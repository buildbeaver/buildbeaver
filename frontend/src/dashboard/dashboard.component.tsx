import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { RequireAuth } from '../components/require-auth/require-auth.component';
import { LegalEntitiesProvider } from '../contexts/legal-entities/legal-entities.provider';
import { Header } from '../components/header/header.component';
import { BuildsBase } from './builds/builds-base.component';
import { OrgsBase } from './orgs/orgs-base.component';
import { UsersBase } from './users/users-base.component';
import { SelectedLegalEntityProvider } from '../contexts/selected-legal-entity/selected-legal-entity.provider';
import { CurrentLegalEntityProvider } from '../contexts/current-legal-entity/current-legal-entity.provider';
import { PollingBannerProvider } from '../contexts/polling-banner/polling-banner.provider';
import { ErrorBoundary } from '../components/error-boundary/error-boundary.component';
import { SetupProvider } from '../contexts/setup/setup.provider';

export function Dashboard(): JSX.Element {
  return (
    <RequireAuth>
      <LegalEntitiesProvider>
        <SelectedLegalEntityProvider>
          <CurrentLegalEntityProvider>
            <PollingBannerProvider>
              <SetupProvider>
                <div className="flex w-full flex-1 flex-col">
                  <Header />
                  <ErrorBoundary>
                    <Routes>
                      <Route path="orgs/:legal_entity_name/*" element={<OrgsBase />} />
                      <Route path="users/:legal_entity_name/*" element={<UsersBase />} />
                      <Route path="builds/*" element={<BuildsBase />} />
                      <Route path="*" element={<Navigate to={'/builds'} replace />} />
                    </Routes>
                  </ErrorBoundary>
                </div>
              </SetupProvider>
            </PollingBannerProvider>
          </CurrentLegalEntityProvider>
        </SelectedLegalEntityProvider>
      </LegalEntitiesProvider>
    </RequireAuth>
  );
}
