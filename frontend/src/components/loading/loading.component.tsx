import React from 'react';
import { BiLoader } from 'react-icons/bi';

interface Props {
  message?: string;
}

export function Loading(props: Props): JSX.Element {
  return (
    <div className="flex h-full w-full items-center justify-center" data-testid="loading">
      <BiLoader className="animate-spin" size={22} />
      {props.message && <div className={'pl-2'}>{props.message}</div>}
    </div>
  );
}
