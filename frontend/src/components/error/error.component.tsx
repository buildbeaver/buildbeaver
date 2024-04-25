import React from 'react';
import { IoCloseCircleSharp } from 'react-icons/io5';

interface Props {
  errorMessage: string;
}

/**
 * Provides common styling for displaying error messages.
 */
export function Error(props: Props): JSX.Element {
  const { errorMessage } = props;

  return (
    <div className="flex w-full items-center gap-x-2 rounded-md border-amaranth bg-amaranthTransparent p-4 text-amaranth">
      <IoCloseCircleSharp size={24} />
      <span className="font-bold">{errorMessage}</span>
    </div>
  );
}
