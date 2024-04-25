import { useEffect, useState } from 'react';
import { fetchBuild } from '../../services/builds.service';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { sortBuild } from '../../utils/build-sort/build-sort.utils';
import { Status } from '../../enums/status.enum';
import { usePolling } from '../polling/polling.hook';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { patchSkippedStatuses } from '../../utils/build.utils';

const runningStates = new Set([Status.Submitted, Status.Queued, Status.Running]);

export interface IUseBuild {
  buildGraph?: IBuildGraph;
  buildGraphError?: IStructuredError;
}

export function useBuild(url: string): IUseBuild {
  const [buildGraph, setBuildGraph] = useState<IBuildGraph | undefined>();
  const [buildGraphError, setBuildGraphError] = useState<IStructuredError | undefined>();

  useEffect(() => {
    setBuildGraph(undefined);
  }, [url]);

  usePolling<IBuildGraph>({
    fetch: () => fetchBuild(url),
    onSuccess: (response) => {
      setBuildGraphError(undefined);
      setBuildGraph(patchSkippedStatuses(sortBuild(response)));
    },
    onError: (error: IStructuredError) => {
      if (!buildGraph) {
        // Only set error if the initial fetch failed, not if subsequent polling fails.
        // The polling banner will handle notifying the user of subsequent polling failures.
        setBuildGraphError(error);
      }
    },
    options: {
      dependencies: [url],
      shouldTick: (response: IBuildGraph) => {
        return runningStates.has(response.build.status);
      }
    }
  });

  return {
    buildGraph: buildGraph,
    buildGraphError: buildGraphError
  };
}
