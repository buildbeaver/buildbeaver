import React from 'react';
import { Handle, Position } from 'reactflow';
import { BuildStatusIndicator } from '../../../dashboard/build-status-indicator/build-status-indicator.component';
import { INodeData } from '../interfaces/node-data.interface';
import { Timer } from '../../timer/timer.component';

interface Props {
  data: INodeData;
}

export function StepNode(props: Props): JSX.Element {
  const { hideSourceHandle, hideTargetHandle, label, status, timings } = props.data;

  return (
    <div className="flex h-full items-center gap-x-3 bg-white px-3">
      <div className="flex w-full justify-between gap-x-2">
        <div className="flex min-w-0 gap-x-1">
          <div>
            <BuildStatusIndicator status={status} size={24} />
          </div>
          <span className="... truncate" title={label}>
            {label}
          </span>
        </div>
        <Timer className="text-gray-400" timings={timings} />
      </div>
      <Handle
        type="target"
        className={`left-[6px] top-[20px] ${hideTargetHandle && 'invisible'}`}
        position={Position.Top}
        isConnectable={false}
      />
      <Handle
        type="source"
        className={`left-[6px] bottom-[20px] ${hideSourceHandle && 'invisible'}`}
        position={Position.Bottom}
        isConnectable={false}
      />
    </div>
  );
}
