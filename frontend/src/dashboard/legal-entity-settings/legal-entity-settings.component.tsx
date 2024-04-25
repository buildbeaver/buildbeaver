import React from 'react';
import { INavigationTab } from '../../interfaces/navigation-tab.interface';
import { TabbedNavigation } from '../../components/tabbed-navigation/tabbed-navigation.component';
import { ViewCentered } from '../../components/view-centered/view-centered.component';

interface Props {
  tabs: INavigationTab[];
}

export function LegalEntitySettings(props: Props): JSX.Element {
  return (
    <ViewCentered navigationTabs={[]}>
      <TabbedNavigation tabs={props.tabs} />
    </ViewCentered>
  );
}
