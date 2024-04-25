import React, { useContext, useMemo } from 'react';
import { Loading } from '../../components/loading/loading.component';
import { BuildTree } from '../build-tree/build-tree.component';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { LogViewer } from '../log-viewer/log-viewer.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { BuildMetadata } from '../build-metadata/build-metadata.component';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ErrorBoundary } from '../../components/error-boundary/error-boundary.component';
import { useIndirectedJobGraph } from '../../hooks/indirected-job-graph/indirected-job.hook';
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

export function BuildLog(props: Props): JSX.Element {
  const { buildGraph, buildGraphError, selectedJobName, jobSelected } = props;
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { build_name, repo_name } = useParams();
  const selectedJobGraph = buildGraph?.jobs?.find((jGraph) => jobFQN(jGraph.job) === selectedJobName);
  const indirectedJobGraphUrl = selectedJobGraph?.job?.indirect_job_url
    ? `${selectedJobGraph.job.indirect_job_url}/graph`
    : undefined;
  const { indirectedJobGraph } = useIndirectedJobGraph(indirectedJobGraphUrl);
  const navigationTabs = useMemo(
    () => makeBuildNavigationTabs(BuildTab.Log, currentLegalEntity, repo_name!, build_name!, selectedJobName),
    [selectedJobName]
  );

  if (buildGraph && (!buildGraph.jobs || buildGraph.jobs.length === 0)) {
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
                <LogViewer jobGraph={selectedJobGraph} indirectedJobGraph={indirectedJobGraph} />
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
