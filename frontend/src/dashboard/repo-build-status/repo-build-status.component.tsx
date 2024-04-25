import React from 'react';
import { Status } from '../../enums/status.enum';

interface Props {
  status: Status;
}

/**
 * Indicates the build status of a repo, as determined by the status of the most recent build.
 */
export function RepoBuildStatus(props: Props): JSX.Element {
  const { status } = props;

  let isPending = false;
  let colour: string;
  let statusText: string;

  switch (status) {
    case Status.Canceled:
      colour = 'amaranth';
      statusText = 'Cancelled';
      break;
    case Status.Failed:
      colour = 'amaranth';
      statusText = 'Failed';
      break;
    case Status.Succeeded:
      colour = 'mountainMeadow';
      statusText = 'Succeeded';
      break;
    case Status.Queued:
    case Status.Running:
    case Status.Submitted:
      isPending = true;
      colour = 'tundora';
      statusText = 'Pending';
      break;
    default:
      colour = 'tundora';
      statusText = 'Unknown';
  }

  return (
    <div className="flex" title={`Default branch build status: ${statusText}`}>
      <div
        className={`flex justify-center rounded-md px-3 text-sm bg-${colour}Transparent text-${colour} font-bold ${
          isPending && 'animate-pulse'
        }`}
      >
        {statusText}
      </div>
    </div>
  );
}
