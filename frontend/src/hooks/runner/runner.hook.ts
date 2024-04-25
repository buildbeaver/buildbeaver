import { useEffect, useState } from 'react';
import { fetchRunner } from '../../services/runners.service';
import { IRunner } from '../../interfaces/runner.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface IUseRunner {
  runner?: IRunner;
  runnerError?: IStructuredError;
  runnerLoading: boolean;
}

export function useRunner(url: string): IUseRunner {
  const [runner, setRunner] = useState<IRunner | undefined>();
  const [runnerError, setRunnerError] = useState<IStructuredError | undefined>();
  const [runnerLoading, setRunnerLoading] = useState(true);

  useEffect(() => {
    const runFetchRunner = async (): Promise<void> => {
      setRunnerLoading(true);

      await fetchRunner(url)
        .then((response) => {
          setRunner(response);
        })
        .catch((error: IStructuredError) => {
          setRunnerError(error);
        })
        .finally(() => {
          setRunnerLoading(false);
        });
    };

    runFetchRunner();
  }, [url]);

  return { runner, runnerError, runnerLoading };
}
