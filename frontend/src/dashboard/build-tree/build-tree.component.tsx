import React from 'react';
import { BulletList } from 'react-content-loader';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { BuildStatusIndicator } from '../build-status-indicator/build-status-indicator.component';
import { IJob } from '../../interfaces/job.interface';
import { NavLink, useLocation } from 'react-router-dom';
import { replacePathParts } from '../../utils/path.utils';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { PlatformIndicator } from '../platform-indicator/platform-indicator.component';
import { Sidebar } from '../../components/sidebar/sidebar.component';
import { jobFQN } from '../../utils/job.utils';

interface Props {
  buildGraph?: IBuildGraph;
  buildGraphError?: IStructuredError;
  selectedJob?: IJob;
  jobSelected: (jobName: string) => void;
}

export function BuildTree(props: Props): JSX.Element {
  const { buildGraph, buildGraphError, selectedJob, jobSelected } = props;
  const location = useLocation();

  const renderJob = (job: IJob): JSX.Element => {
    const isSelected = job.id === selectedJob?.id;

    return (
      <NavLink
        key={job.id}
        className={`flex cursor-pointer items-center gap-x-1 rounded-md px-2 py-1 hover:bg-gray-100 ${
          isSelected && 'bg-gray-100'
        }`}
        onClick={() => jobSelected(jobFQN(job))}
        to={replacePathParts(location.pathname, [{ positionFromEnd: 1, replacement: jobFQN(job) }])}
      >
        <div>
          <BuildStatusIndicator status={job.status} size={16} />
        </div>
        {job.runs_on && (
          <div>
            <PlatformIndicator runsOn={job.runs_on} size={16} />
          </div>
        )}
        <span className={`... truncate ${isSelected && 'font-bold'}`} title={jobFQN(job)}>
          {jobFQN(job)}
        </span>
      </NavLink>
    );
  };

  return (
    <Sidebar>
      {!buildGraph && !buildGraphError && <BulletList />}
      {buildGraph && !buildGraphError && buildGraph.jobs?.map((jGraph) => renderJob(jGraph.job))}
    </Sidebar>
  );
}
