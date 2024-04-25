import React from 'react';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { getStructuredErrorMessage } from '../../utils/error.utils';

interface Props {
  error: IStructuredError;
  isFocused: boolean;
  isLast: boolean;
}

export function FailedSearchResult(props: Props): JSX.Element {
  const { error, isFocused, isLast } = props;
  const errorMessage = getStructuredErrorMessage(error, 'Failed to load search result');

  return (
    <div
      className={`bg-amaranthTransparent p-2 text-amaranth ${isLast ? 'rounded-b-md' : 'border-b'} ${isFocused && 'bg-blue-100'}`}
    >
      {errorMessage}
    </div>
  );
}
