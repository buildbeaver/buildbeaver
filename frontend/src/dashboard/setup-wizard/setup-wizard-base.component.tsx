import React from 'react';
import { ViewCentered } from '../../components/view-centered/view-centered.component';
import { SetupWizard } from './setup-wizard.component';
import { TickProvider } from '../../contexts/tick/tick.provider';

export function SetupWizardBase(): JSX.Element {
  return (
    <ViewCentered navigationTabs={[]}>
      <TickProvider>
        <SetupWizard />
      </TickProvider>
    </ViewCentered>
  );
}
