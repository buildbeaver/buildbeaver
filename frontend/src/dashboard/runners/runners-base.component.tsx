import React, { useContext, useMemo } from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { RunnersView } from './runners-view.component';
import { ViewCentered } from '../../components/view-centered/view-centered.component';
import { RunnerRegister } from '../runner/runner-register.component';
import { RunnerBase } from '../runner/runner-base.component';
import { LegalEntityTab, makeLegalEntityNavigationTabs } from '../../utils/navigation-utils';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';

/**
 * Base component for Runners that handles routes for anything Runner related.
 */
export function RunnersBase(): JSX.Element {
  const { selectedLegalEntity } = useContext(SelectedLegalEntityContext);
  const navigationTabs = useMemo(
    () => makeLegalEntityNavigationTabs(LegalEntityTab.Runners, selectedLegalEntity),
    [selectedLegalEntity]
  );

  return (
    <ViewCentered navigationTabs={navigationTabs}>
      <Routes>
        <Route path=":runner_name/*" element={<RunnerBase />} />
        <Route path="register" element={<RunnerRegister />} />
        <Route path="/" element={<RunnersView />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </ViewCentered>
  );
}
