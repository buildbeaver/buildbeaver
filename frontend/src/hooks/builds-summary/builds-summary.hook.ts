import { useEffect, useState } from 'react';
import { fetchBuildSummary } from '../../services/legal-entity.service';
import { IBuildsSummary } from '../../interfaces/builds-summary.interface';
import { usePolling } from '../polling/polling.hook';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface IUseBuildsSummary {
  buildsSummary?: IBuildsSummary;
  buildsSummaryError?: IStructuredError;
}

export function useBuildsSummary(url: string): IUseBuildsSummary {
  const [buildsSummary, setBuildsSummary] = useState<IBuildsSummary | undefined>();
  const [buildsSummaryError, setBuildsSummaryError] = useState<IStructuredError | undefined>();

  usePolling<IBuildsSummary>({
    fetch: () => fetchBuildSummary(url),
    onSuccess: (response) => {
      setBuildsSummaryError(undefined);
      setBuildsSummary(response);
    },
    onError: (error: IStructuredError) => {
      if (!buildsSummary) {
        // Only set error if the initial fetch failed, not if subsequent polling fails.
        // The polling banner will handle notifying the user of subsequent polling failures.
        setBuildsSummaryError(error);
      }
    },
    options: {
      dependencies: [url]
    }
  });

  useEffect(() => {
    setBuildsSummary(undefined);
    setBuildsSummaryError(undefined);
  }, [url]);

  return {
    buildsSummary,
    buildsSummaryError
  };
}
