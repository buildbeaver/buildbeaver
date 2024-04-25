import { isObject } from 'lodash';
import { useEffect, useRef, useState } from 'react';
import { usePolling } from '../polling/polling.hook';
import { apiGet, apiPost } from '../../services/api.service';
import { IResourceResponse } from '../../services/responses/resource-response.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';

/**
 * IUseLiveResources describes the interaction of an API call with a resource
 * endpoint.
 */
export interface IUseLiveResources<Type> {
  // Set if an error has occurred during the API interaction.
  error?: IStructuredError;
  // True if the API call is currently in progress.
  loading: boolean;
  // If populated, response provides the result of the API call.
  response?: IResourceResponse<Type>;
}

/**
 * IUseStaticResources describes the interaction of an API call with a resource
 * endpoint with the ability to refresh the call.
 */
export interface IUseStaticResources<Type> extends IUseLiveResources<Type> {
  refresh: () => void;
}

/**
 * ResourceRequest describes a request to our resource API.
 */
export interface ResourceRequest {
  query?: object; // TODO pretty utils for generating filters
  url: string;
}

/**
 * useLiveResourceList provides a way to request a resource via our API with
 * automated polling in-effect over the API call.
 */
export function useLiveResourceList<Type>(request: ResourceRequest): IUseLiveResources<Type> {
  const [response, setResponse] = useState<IResourceResponse<Type> | undefined>();
  const [error, setError] = useState<IStructuredError | undefined>();
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    setLoading(true);
    setError(undefined);
    setResponse(undefined);
  }, [request.url]);

  usePolling<IResourceResponse<Type>>({
    fetch: () =>
      isObject(request.query)
        ? apiPost<IResourceResponse<Type>>(request.url, request.query)
        : apiGet<IResourceResponse<Type>>(request.url),
    onSuccess: (response) => {
      successHandler(response, { setError, setResponse, setLoading });
    },
    onError: (error: IStructuredError) => {
      setLoading(false);
      if (!response) {
        // Only set error if the initial fetch failed, not if subsequent polling fails.
        // The polling banner will handle notifying the user of subsequent polling failures.
        setError(error);
      }
    },
    options: { dependencies: [request.url] }
  });

  return {
    response,
    error,
    loading
  };
}

/**
 * useStaticResourceList provides a way to request a resource via our API with
 * the ability to manually refresh the response.
 */
export function useStaticResourceList<Type>(request: ResourceRequest): IUseStaticResources<Type> {
  const [tick, setTick] = useState(true);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<IStructuredError | undefined>();
  const [response, setResponse] = useState<IResourceResponse<Type> | undefined>();
  const initialFetchComplete = useRef(false);

  const triggerRefresh = () => {
    // Components that call triggerRefresh() inside a useEffect() hook will automatically call triggerRefresh() on
    // the initial component render. Block any calls to triggerRefresh() here until we have attempted to fetch data
    // once to prevent two identical api calls being made on initial component render.
    if (initialFetchComplete.current) {
      setLoading(true);
      setTick(!tick);
    }
  };

  useEffect(() => {
    const runFetch = async (): Promise<void> => {
      await (isObject(request.query)
        ? apiPost<IResourceResponse<Type>>(request.url, request.query)
        : apiGet<IResourceResponse<Type>>(request.url)
      )
        .then((response) => {
          successHandler(response, { setError, setResponse, setLoading });
        })
        .catch((error: IStructuredError) => {
          setError(error);
        })
        .finally(() => {
          setLoading(false);
          initialFetchComplete.current = true;
        });
    };

    runFetch();
  }, [request.url, tick]);

  useEffect(() => {
    setLoading(true);
  }, [request.url]);

  return {
    loading,
    error,
    response,
    refresh: triggerRefresh
  };
}

/**
 * Common success handler for either a static or live request for ensuring that state is handled
 * in a consistent manner.
 *
 * Ensures that the response.results array is empty if we have nothing from the API.
 * @param response the returned response from our API.
 * @param stateHandler
 */
function successHandler<Type>(response: IResourceResponse<Type>, stateHandler: requestStateHandler<Type>) {
  stateHandler.setError(undefined);
  if (response && response.results == null) {
    response.results = new Array<Type>();
  }
  stateHandler.setResponse(response);
  stateHandler.setLoading(false);
}

/**
 * requestStateHandler provides a wrapper around the state logic held within
 * useLiveResourceList and useStaticResourceList so that we can provide a
 * consistent handler for our response parsing.
 */
interface requestStateHandler<Type> {
  setError: (error: IStructuredError | undefined) => void;
  setLoading: (loading: boolean) => void;
  setResponse: (response: IResourceResponse<Type> | undefined) => void;
}
