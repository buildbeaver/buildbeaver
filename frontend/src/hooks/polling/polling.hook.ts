import { useContext, useEffect, useState } from 'react';
import { getRandomInteger } from '../../utils/build-sort/math.utils';
import { PollingBannerContext } from '../../contexts/polling-banner/polling-banner.context';
import { PollingConnection } from '../../models/polling-connection.model';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { DateTime } from 'luxon';

interface IPollingOptions<TFetchResponse> {
  /**
   * Changes in this array will re-initialise polling.
   */
  dependencies: any[];

  /**
   * Polling will be abandon if we exceed this many failures consecutively.
   */
  maxRetries?: number;

  /**
   * Return false in this callback to manually abandon polling.
   */
  shouldTick?: (response: TFetchResponse) => boolean;

  /**
   * When true the polling banner will be used to show the user the global polling connection status.
   */
  usePollingBanner?: boolean;
}

class PollingCancellationToken {
  isCancelled = false;

  cancel(): void {
    this.isCancelled = true;
  }
}

class PollingRequest<TFetchResponse> {
  fetch: () => Promise<TFetchResponse>;
  onSuccess: (response: TFetchResponse) => void;
  onError: (error: IStructuredError) => void;
  onFinally?: () => void = () => {};
  options?: IPollingOptions<TFetchResponse> = defaultPollingOptions;
}

const defaultPollingOptions = {
  dependencies: [],
  maxRetries: 30,
  shouldTick: () => true,
  usePollingBanner: true
};

export function usePolling<TFetchResponse>(pollingRequest: PollingRequest<TFetchResponse>): void {
  const [tick, setTick] = useState(DateTime.now().toMillis());
  const { addPollingConnection } = useContext(PollingBannerContext);
  const [retries, setRetries] = useState(0);
  const [pollingConnection, setPollingConnection] = useState<PollingConnection | undefined>();
  const [pollingCancellationToken, setPollingCancellationToken] = useState<PollingCancellationToken | undefined>();

  const { fetch, onSuccess, onError, onFinally, options } = pollingRequest;
  const { dependencies, maxRetries, shouldTick, usePollingBanner } = {
    ...defaultPollingOptions,
    ...options
  };

  /**
   * Abandons the last referenced polling connection so the polling banner shows that the connection was lost.
   */
  const abandonPollingConnection = (): void => {
    pollingConnection?.abandon();
  };

  /**
   * Restores the last referenced polling connection so the polling banner will be hidden.
   */
  const clearPollingConnection = (): void => {
    if (usePollingBanner && pollingConnection) {
      pollingConnection.restore();
      setPollingConnection(undefined);
    }
  };

  /**
   * Resets retries to 0 when polling succeeds. We only want to abandon polling in the event of consecutive errors.
   */
  const clearRetries = (): void => {
    setRetries(0);
  };

  /**
   * Creates and maintains a reference to a new polling connection if we are not already retrying.
   */
  const initPollingConnection = (): void => {
    const isFirstError = !pollingConnection;

    if (usePollingBanner && isFirstError) {
      const pollingConnection = new PollingConnection();
      setPollingConnection(pollingConnection);
      addPollingConnection(pollingConnection);
    }
  };

  const onPollError = (error: IStructuredError, pollingIntervalMs: number): void => {
    const isNetworkError = error.message === 'Network request failed';

    onError(error);

    if (error.statusCode === 403 || error.statusCode === 404) {
      return;
    }

    if (isNetworkError) {
      initPollingConnection();
    }

    if (retries < maxRetries) {
      setTimeout(() => {
        setTick(DateTime.now().toMillis());
        setRetries(retries + 1);
      }, pollingIntervalMs);
    } else if (isNetworkError) {
      abandonPollingConnection();
    }
  };

  const onPollSuccess = (response: TFetchResponse, pollingIntervalMs: number): void => {
    onSuccess(response);
    clearRetries();
    clearPollingConnection();

    if (shouldTick(response)) {
      setTimeout(() => {
        setTick(DateTime.now().toMillis());
        setRetries(0);
      }, pollingIntervalMs);
    }
  };

  const runIfNotCancelled = (token: PollingCancellationToken, onPollHandler: () => void): void => {
    if (!token.isCancelled) {
      onPollHandler();
    }
  };

  useEffect(() => {
    const token = new PollingCancellationToken();

    // This effect can run at erratic intervals if the dependencies passed in by the component are changed. When this
    // occurs, any in progress fetches are considered stale and cancellation should be requested. The simplest solution
    // here is to always cancel the previous token.
    pollingCancellationToken?.cancel();
    setPollingCancellationToken(token);

    const poll = async (): Promise<void> => {
      const pollingIntervalMs = getRandomInteger(3000, 5000);

      await fetch()
        .then((response: TFetchResponse) => runIfNotCancelled(token, () => onPollSuccess(response, pollingIntervalMs)))
        .catch((error: IStructuredError) => runIfNotCancelled(token, () => onPollError(error, pollingIntervalMs)))
        .finally(() => onFinally && onFinally());
    };

    poll();
  }, [tick].concat(dependencies));
}
