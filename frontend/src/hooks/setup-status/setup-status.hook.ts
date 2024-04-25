import { useEffect, useState } from 'react';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ISetupStatus } from '../../interfaces/setup-status.interface';
import { fetchSetupStatus } from '../../services/legal-entity.service';
import { DateTime } from 'luxon';
import { usePolling } from '../polling/polling.hook';

export interface IUseSetupStatus {
  setupStatus?: ISetupStatus;
  setupStatusError?: IStructuredError;
  setupStatusLoading: boolean;
  setupStatusRefreshing: boolean;
  refreshSetupStatus: () => void;
}

export function useSetupStatus(url: string, poll = false): IUseSetupStatus {
  const [setupStatus, setSetupStatus] = useState<ISetupStatus | undefined>();
  const [setupStatusError, setSetupStatusError] = useState<IStructuredError | undefined>();
  const [setupStatusLoading, setSetupStatusLoading] = useState(true);
  const [setupStatusRefreshing, setSetupStatusRefreshing] = useState(false);
  const [tick, setTick] = useState(DateTime.now().toMillis());

  usePolling<ISetupStatus>({
    fetch: () => fetchSetupStatus(url),
    onSuccess: (response) => {
      setSetupStatusError(undefined);
      setSetupStatus(response);
    },
    onError: (error: IStructuredError) => {
      setSetupStatusError(error);
    },
    onFinally: () => {
      setSetupStatusLoading(false);
      setSetupStatusRefreshing(false);
    },
    options: {
      dependencies: [url, tick, poll],
      shouldTick: () => poll
    }
  });

  useEffect(() => {
    setSetupStatusLoading(true);
  }, [url]);

  const refreshSetupStatus = (): void => {
    setSetupStatusRefreshing(true);
    setTick(DateTime.now().toMillis());
  };

  return { setupStatus, setupStatusError, setupStatusLoading, setupStatusRefreshing, refreshSetupStatus };
}
