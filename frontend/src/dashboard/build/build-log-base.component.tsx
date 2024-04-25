import React from 'react';
import { BuildLog } from './build-log.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { BuildBase } from './build-base.component';

export function BuildLogBase(): JSX.Element {
  const getElement = (
    buildGraph: IBuildGraph | undefined,
    buildGraphError: IStructuredError | undefined,
    selectedJobName: string | undefined,
    jobSelected: (jobName: string) => void
  ): JSX.Element => {
    return (
      <BuildLog
        buildGraph={buildGraph}
        buildGraphError={buildGraphError}
        selectedJobName={selectedJobName}
        jobSelected={jobSelected}
      />
    );
  };

  return <BuildBase path="log" getElement={getElement} />;
}
