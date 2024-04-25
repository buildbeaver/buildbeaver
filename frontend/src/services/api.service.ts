import { fetch as crossFetch } from 'cross-fetch';
import { IStructuredError } from '../interfaces/structured-error.interface';

export async function apiDelete(url: string, options?: RequestInit): Promise<void> {
  options = {
    ...options,
    method: 'DELETE'
  };

  return await fetch(url, options);
}

export async function apiGet<T = unknown>(url: string, options?: RequestInit): Promise<T> {
  return await fetch(url, options);
}

export async function apiPost<T = unknown>(url: string, body: object, options?: RequestInit): Promise<T> {
  options = {
    ...options,
    body: JSON.stringify(body),
    method: 'POST'
  };

  return await fetch(url, options);
}

export async function apiPatch<T = unknown>(url: string, body: object, options?: RequestInit): Promise<T> {
  options = {
    ...options,
    body: JSON.stringify(body),
    method: 'PATCH'
  };

  return await fetch(url, options);
}

async function fetch<T>(url: string, options?: RequestInit): Promise<T> {
  options = options || {};

  return crossFetch(url, {
    ...options,
    credentials: 'include', // https://github.com/github/fetch#sending-cookies
    headers: {
      ...options.headers,
      Accept: 'application/json',
      'Content-Type': 'application/json'
    }
  }).then((response) => {
    if (response.status === 401) {
      window.location.href = '/sign-out';
    }

    if (response.status >= 200 && response.status < 300) {
      if (response.status === 204) {
        return null;
      }

      return response.json();
    }

    const error: IStructuredError = { statusText: response.statusText, statusCode: response.status };

    return response
      .json()
      .catch((jsonError: Error) => {
        // there was an error parsing the server response as JSON, just
        // return what we can to the outside world
        console.error(`Unable to parse server error response as JSON: ${jsonError}`);
        throw error;
      })
      .then((response: any) => {
        // we parsed the server response as JSON, which means this is a structured
        // response with descriptive error message etc.
        error.serverError = response;
        throw error;
      });
  });
}
