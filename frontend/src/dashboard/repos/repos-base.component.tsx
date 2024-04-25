import React, { useContext, useMemo } from 'react';
import { ViewCentered } from '../../components/view-centered/view-centered.component';
import { Repos } from './repos.component';
import { LegalEntityTab, makeLegalEntityNavigationTabs } from '../../utils/navigation-utils';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';

export function ReposBase(): JSX.Element {
  const { selectedLegalEntity } = useContext(SelectedLegalEntityContext);
  const navigationTabs = useMemo(
    () => makeLegalEntityNavigationTabs(LegalEntityTab.Repos, selectedLegalEntity),
    [selectedLegalEntity]
  );

  return (
    <ViewCentered navigationTabs={navigationTabs}>
      <Repos />
    </ViewCentered>
  );
}
