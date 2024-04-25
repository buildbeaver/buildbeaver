import React from 'react';
import { BuildStatusIndicator } from '../../../dashboard/build-status-indicator/build-status-indicator.component';
import { Status } from '../../../enums/status.enum';

interface Props {
  data: {
    message: string;
  };
}

export function ErrorNode(props: Props): JSX.Element {
  return (
    <div className="flex h-full cursor-default flex-col rounded border-2 border-amaranth bg-amaranthTransparent">
      <div className="flex h-full items-center gap-x-3 bg-white p-4">
        <BuildStatusIndicator status={Status.Failed} />
        {props.data.message}
      </div>
    </div>
  );
}
