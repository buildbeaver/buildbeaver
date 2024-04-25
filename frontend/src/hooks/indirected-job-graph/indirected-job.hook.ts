import { useState } from 'react';
import { Status } from '../../enums/status.enum';
import { usePolling } from '../polling/polling.hook';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { IJobGraph } from '../../interfaces/job-graph.interface';
import { fetchJobGraph } from '../../services/jobs.service';
import { sortJob } from '../../utils/build-sort/build-sort.utils';

const runningStates = new Set([Status.Submitted, Status.Queued, Status.Running]);

export interface IUseIndirectedJob {
  indirectedJobGraph?: IJobGraph;
  indirectedJobGraphError?: IStructuredError;
}

export function useIndirectedJobGraph(url?: string): IUseIndirectedJob {
  const [indirectedJobGraph, setIndirectedJobGraph] = useState<IJobGraph | undefined>();
  const [indirectedJobGraphError, setIndirectedJobGraphError] = useState<IStructuredError | undefined>();

  usePolling<IJobGraph | undefined>({
    fetch: () => (url ? fetchJobGraph(url) : Promise.resolve(undefined)),
    onSuccess: (response) => {
      setIndirectedJobGraphError(undefined);

      if (response) {
        setIndirectedJobGraph(sortJob(response));
      } else {
        setIndirectedJobGraph(response);
      }
    },
    onError: (error: IStructuredError) => {
      if (!indirectedJobGraph) {
        // Only set error if the initial fetch failed, not if subsequent polling fails.
        // The polling banner will handle notifying the user of subsequent polling failures.
        setIndirectedJobGraphError(error);
      }
    },
    options: {
      dependencies: [url],
      shouldTick: (response: IJobGraph | undefined) => {
        return !!response && runningStates.has(response.job.status);
      }
    }
  });

  return {
    indirectedJobGraph,
    indirectedJobGraphError
  };
}
