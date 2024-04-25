import React, { useContext, useMemo } from 'react';
import { Loading } from '../../components/loading/loading.component';
import { BuildFlow } from '../../components/build-flow/build-flow.component';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { BuildMetadata } from '../build-metadata/build-metadata.component';
import { ErrorBoundary } from '../../components/error-boundary/error-boundary.component';
import { ViewFullWidth } from '../../components/view-full-width/view-full-width.component';
import { useBuild } from '../../hooks/build/build.hook';
import { useBuildUrl } from '../../hooks/build-url/build-url.hook';
import { BuildTab, makeBuildNavigationTabs } from '../../utils/navigation-utils';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { useParams } from 'react-router-dom';
import { EmptyBuild } from './empty-build.component';

export function BuildGraph(): JSX.Element {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { build_name, repo_name } = useParams();
  const buildUrl = useBuildUrl();
  const { buildGraph, buildGraphError } = useBuild(buildUrl);
  const navigationTabs = useMemo(() => makeBuildNavigationTabs(BuildTab.Graph, currentLegalEntity, repo_name!, build_name!), []);

  if (buildGraph && (!buildGraph.jobs || buildGraph.jobs.length === 0)) {
    return <EmptyBuild buildGraph={buildGraph} navigationTabs={navigationTabs} />;
  }

  return (
    <ViewFullWidth navigationTabs={navigationTabs}>
      <div className="flex h-full flex-col">
        {buildGraph && <BuildMetadata buildGraph={buildGraph} />}
        <div className="my-4 flex h-full flex-col">
          <ErrorBoundary>
            {buildGraph && <BuildFlow bGraph={buildGraph} />}
            {buildGraphError && <StructuredError error={buildGraphError} handleNotFound={true} />}
            {!buildGraph && !buildGraphError && <Loading />}
          </ErrorBoundary>
        </div>
      </div>
    </ViewFullWidth>
  );
}
