import React from 'react';
import { BuildArtifacts } from './build-artifacts.component';
import { BuildBase } from './build-base.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';

export function BuildArtifactsBase(): JSX.Element {
  const getElement = (
    buildGraph: IBuildGraph | undefined,
    buildGraphError: IStructuredError | undefined,
    selectedJobName: string | undefined,
    jobSelected: (jobName: string) => void
  ): JSX.Element => {
    return (
      <BuildArtifacts
        buildGraph={buildGraph}
        buildGraphError={buildGraphError}
        selectedJobName={selectedJobName}
        jobSelected={jobSelected}
      />
    );
  };

  return <BuildBase path="artifacts" getElement={getElement} />;
}
