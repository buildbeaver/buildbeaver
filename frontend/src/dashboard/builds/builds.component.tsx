import React, { useContext, useMemo } from 'react';
import { BuildsSidebar } from '../builds-sidebar/builds-sidebar.component';
import { useBuildsSummary } from '../../hooks/builds-summary/builds-summary.hook';
import { SectionHeading } from '../../components/section-heading/section-heading.component';
import { BuildsList } from '../builds-list/builds-list.component';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { LegalEntityTab, makeLegalEntityNavigationTabs } from '../../utils/navigation-utils';
import { ViewFullWidth } from '../../components/view-full-width/view-full-width.component';

/**
 * A view showing details of multiple builds.
 */
export function Builds(): JSX.Element {
  const { selectedLegalEntity } = useContext(SelectedLegalEntityContext);
  const { buildsSummary, buildsSummaryError } = useBuildsSummary(selectedLegalEntity.build_summary_url);
  const navigationTabs = useMemo(
    () => makeLegalEntityNavigationTabs(LegalEntityTab.Builds, selectedLegalEntity),
    [selectedLegalEntity]
  );

  const section = (heading: string, builds?: IBuildGraph[] | undefined): JSX.Element => {
    return (
      <React.Fragment>
        <SectionHeading text={heading} />
        {buildsSummary || buildsSummaryError ? (
          <BuildsList builds={builds} error={buildsSummaryError} />
        ) : (
          <SimpleContentLoader rowHeight={40} numberOfRows={3} />
        )}
      </React.Fragment>
    );
  };

  return (
    <ViewFullWidth navigationTabs={navigationTabs}>
      <div className="flex h-full gap-x-6">
        <BuildsSidebar />
        <div className="flex grow flex-col gap-y-6">
          {section('Running', buildsSummary?.running)}
          {section('Upcoming', buildsSummary?.upcoming)}
          {section('Completed', buildsSummary?.completed)}
        </div>
      </div>
    </ViewFullWidth>
  );
}
