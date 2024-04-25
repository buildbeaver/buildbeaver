import React from 'react';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { getStructuredErrorMessage } from '../../utils/error.utils';

interface Props {
  error: IStructuredError;
  fallback?: string;
}

/**
 * A small wrapper around getStructuredErrorMessage() to save us from calling it from various templates
 */
export function StructuredErrorMessage(props: Props): JSX.Element {
  const { error, fallback } = props;
  const errorMessage = getStructuredErrorMessage(error, fallback);

  return <>{errorMessage}</>;
}
