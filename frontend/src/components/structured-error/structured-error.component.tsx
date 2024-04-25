import React from 'react';
import { Navigate } from 'react-router-dom';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { NotFound } from '../not-found/not-found.component';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { Error } from '../error/error.component';

interface Props {
  error: IStructuredError;
  fallback?: string;
  handleNotFound?: boolean;
}

/**
 * Renders server generated structured errors as user facing error messages.
 */
export function StructuredError(props: Props): JSX.Element {
  const { error, fallback, handleNotFound } = props;
  const { statusCode } = error;

  if (statusCode === 401) {
    return <Navigate to="/sign-out" />;
  }

  if (handleNotFound && statusCode === 404) {
    return <NotFound />;
  }

  const errorMessage = getStructuredErrorMessage(error, fallback);

  return <Error errorMessage={errorMessage} />;
}
