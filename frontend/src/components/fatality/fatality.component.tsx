import React from 'react';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { StructuredError } from '../structured-error/structured-error.component';

interface Props {
  error: IStructuredError;
  message?: string;
}

export function Fatality(props: Props): JSX.Element {
  const { error, message } = props;
  const fallback = message || 'BuildBeaver is unavailable. Please try again later.';

  return (
    <div className="flex w-full items-center justify-center gap-x-2 p-2">
      <div className="flex-1"></div>
      <div className="flex-1">
        <StructuredError error={error} fallback={fallback} />
      </div>
      <div className="flex-1"></div>
    </div>
  );
}
