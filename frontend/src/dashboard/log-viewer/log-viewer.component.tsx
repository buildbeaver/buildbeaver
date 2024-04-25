import { IJobGraph } from '../../interfaces/job-graph.interface';
import React from 'react';
import { LogGroup } from '../log-group/log-group.component';
import { Status } from '../../enums/status.enum';
import { isFinished } from '../../utils/build-status.utils';
import { LogViewerMenu } from './log-viewer-menu.component';
import { jobFQN } from '../../utils/job.utils';

interface Props {
  indirectedJobGraph?: IJobGraph;
  jobGraph: IJobGraph;
}

export function LogViewer(props: Props): JSX.Element {
  const { indirectedJobGraph, jobGraph } = props;
  const forceCollapseSetUpJob = jobGraph.steps.some((step) => step.status === Status.Running);
  const steps = indirectedJobGraph?.steps ?? jobGraph.steps;
  const isJobFinished = isFinished(jobGraph.job.status);

  return (
    <div className="flex grow flex-col rounded-md border bg-gray-800 p-2 text-sm text-white shadow-md">
      <div className="relative flex flex-col">
        <div className="sticky top-0 z-[2] bg-gray-800 p-2">
          <div className="flex justify-between gap-x-2">
            <span className="... truncate font-bold" title={jobFQN(jobGraph.job)}>
              {jobFQN(jobGraph.job)}
            </span>
            {isJobFinished && <LogViewerMenu jobGraph={jobGraph} />}
          </div>
          <hr className="mt-2 border-gray-500" />
        </div>
        <LogGroup
          forceCollapse={forceCollapseSetUpJob}
          id={jobGraph.job.id}
          logDescriptorUrl={jobGraph.job.log_descriptor_url}
          name={indirectedJobGraph ? 'overview... skipped' : 'overview'}
          status={jobGraph.job.status}
          timings={jobGraph.job.timings}
        />
        {indirectedJobGraph && (
          <LogGroup
            forceCollapse={false}
            id={indirectedJobGraph.job.id}
            logDescriptorUrl={indirectedJobGraph.job.log_descriptor_url}
            name="overview"
            status={indirectedJobGraph.job.status}
            timings={indirectedJobGraph.job.timings}
          />
        )}
        {steps.map((step) => (
          <div key={step.id}>
            <LogGroup
              error={step.error}
              forceCollapse={false}
              id={step.id}
              logDescriptorUrl={step.log_descriptor_url}
              name={step.name}
              status={step.status}
              timings={step.timings}
            />
          </div>
        ))}
      </div>
    </div>
  );
}
