import React from 'react';
import { Handle, Position } from 'reactflow';
import { INodeData } from '../interfaces/node-data.interface';
import { getColourForStatus } from '../../../utils/build-status.utils';
import { Timer } from '../../timer/timer.component';
import { PlatformIndicator } from '../../../dashboard/platform-indicator/platform-indicator.component';

interface Props {
  data: INodeData;
}

export function JobNode(props: Props): JSX.Element {
  const { hideSourceHandle, hideTargetHandle, label, runsOn, status, timings } = props.data;
  const statusColour = getColourForStatus(status);

  return (
    <div className={`flex cursor-pointer flex-col bg-${statusColour}Transparent h-full rounded`}>
      <div className="flex justify-between gap-x-2 rounded-t border-b-2 bg-white p-2 px-4 text-lg">
        <div className="flex min-w-0 items-center gap-x-1">
          {runsOn && (
            <div>
              <PlatformIndicator runsOn={runsOn} />
            </div>
          )}
          <span className="... truncate font-bold" title={label}>
            {label}
          </span>
        </div>
        <Timer className="text-gray-400" timings={timings} />
      </div>
      <Handle type="target" className={`${hideTargetHandle && 'invisible'}`} isConnectable={false} position={Position.Left} />
      <Handle type="source" className={`${hideSourceHandle && 'invisible'}`} isConnectable={false} position={Position.Right} />
    </div>
  );
}
