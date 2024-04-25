import React, { useContext, useMemo } from 'react';
import { Loading } from '../../components/loading/loading.component';
import { BuildTree } from '../build-tree/build-tree.component';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { ArtifactViewer } from '../artifact-viewer/artifact-viewer.component';
import { BuildMetadata } from '../build-metadata/build-metadata.component';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ErrorBoundary } from '../../components/error-boundary/error-boundary.component';
import { BuildTab, makeBuildNavigationTabs } from '../../utils/navigation-utils';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { useParams } from 'react-router-dom';
import { EmptyBuild } from './empty-build.component';
import { ViewFullWidth } from '../../components/view-full-width/view-full-width.component';
import { jobFQN } from '../../utils/job.utils';

interface Props {
  buildGraph?: IBuildGraph;
  buildGraphError?: IStructuredError;
  selectedJobName?: string;
  jobSelected: (jobName: string) => void;
}

export function BuildArtifacts(props: Props): JSX.Element {
  const { buildGraph, buildGraphError, selectedJobName, jobSelected } = props;
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { build_name, repo_name } = useParams();
  const selectedJobGraph = buildGraph?.jobs?.find((jGraph) => jobFQN(jGraph.job) === selectedJobName);
  const navigationTabs = useMemo(
    () => makeBuildNavigationTabs(BuildTab.Artifacts, currentLegalEntity, repo_name!, build_name!, selectedJobName),
    [selectedJobName]
  );

  if (buildGraph && !selectedJobGraph) {
    return <EmptyBuild buildGraph={buildGraph} navigationTabs={navigationTabs} />;
  }

  return (
    <ViewFullWidth navigationTabs={navigationTabs}>
      <div className="flex h-full flex-col">
        {buildGraph && <BuildMetadata buildGraph={buildGraph} />}
        <div className="my-4 flex h-full gap-x-4">
          <BuildTree
            buildGraph={buildGraph}
            buildGraphError={buildGraphError}
            selectedJob={selectedJobGraph?.job}
            jobSelected={jobSelected}
          />
          <ErrorBoundary>
            <div className="flex grow flex-col">
              {buildGraph && selectedJobGraph && (
                <ArtifactViewer artifactSearchUri={buildGraph.build.artifact_search_url} selectedJob={selectedJobGraph.job} />
              )}
              {buildGraphError && <StructuredError error={buildGraphError} handleNotFound={true} />}
              {!buildGraph && !buildGraphError && <Loading />}
            </div>
          </ErrorBoundary>
        </div>
      </div>
    </ViewFullWidth>
  );
}
