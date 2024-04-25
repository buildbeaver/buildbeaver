import { useEffect, useState } from 'react';
import { IUseLogs } from './use-logs.interface';
import { fetchLogDescriptor, fetchLogs } from '../../services/logs.service';
import { IStructuredLog } from '../../interfaces/structured-log.interface';
import { LogKind } from '../../enums/log-kind.enum';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { DateTime } from 'luxon';
import { getRandomInteger } from '../../utils/build-sort/math.utils';

export function useLogs(url: string): IUseLogs {
  const [logs, setLogs] = useState<IStructuredLog[] | undefined>();
  const [logsError, setLogsError] = useState<IStructuredError | undefined>();
  const [dataUrl, setDataUrl] = useState<string | undefined>();
  const [continuePolling, setContinuePolling] = useState(true);
  const [start, setStart] = useState(1);
  const [isLoadingLogs, setIsLoadingLogs] = useState(true);
  const [tick, setTick] = useState(DateTime.now().toMillis());

  const updateContinuePolling = (logs: IStructuredLog[]): void => {
    const hasLoggingEnded = logs.length > 0 && logs[logs.length - 1].kind === LogKind.LogEnd;

    if (hasLoggingEnded) {
      setContinuePolling(false);
    }
  };

  const updateStart = (logs: IStructuredLog[]): void => {
    if (logs.length > 0) {
      const nextInSequence = logs[logs.length - 1].seq_no + 1;

      setStart(nextInSequence);
    }
  };

  useEffect(() => {
    setLogs(undefined);
    setLogsError(undefined);
    setDataUrl(undefined);
    setContinuePolling(true);
    setStart(1);
    setIsLoadingLogs(true);
  }, [url]);

  useEffect(() => {
    const runFetchLogs = async (logsUrl: string): Promise<void> => {
      await fetchLogs(logsUrl, start)
        .then((response) => {
          setIsLoadingLogs(false);
          setLogs([...(logs ?? []), ...response]);
          updateStart(response);
          updateContinuePolling(response);

          setTimeout(() => {
            setTick(DateTime.now().toMillis());
          }, getRandomInteger(3000, 5000));
        })
        .catch((error: IStructuredError) => {
          setIsLoadingLogs(false);
          setLogsError(error);
        });
    };

    const runFetch = async (): Promise<void> => {
      if (dataUrl) {
        await runFetchLogs(dataUrl);
      } else {
        await fetchLogDescriptor(url)
          .then((response) => {
            setDataUrl(response.data_url);
          })
          .catch((error: IStructuredError) => {
            setIsLoadingLogs(false);
            setLogsError(error);
          });
      }
    };

    if (continuePolling) {
      runFetch();
    }
  }, [url, tick, dataUrl]);

  return {
    logs,
    logsError,
    isLoadingLogs
  };
}
