import React from 'react';
import { IStructuredLog } from '../../interfaces/structured-log.interface';
import { LogKind } from '../../enums/log-kind.enum';

interface Props {
  children?: React.ReactNode;
  line: IStructuredLog;
}

export function LogLine(props: Props): JSX.Element {
  const { children, line } = props;

  if (line.kind === LogKind.LogEnd) {
    return <></>;
  }

  return (
    <div className="flex grow gap-x-2 rounded py-0.5 text-xs hover:bg-athensTransparent" key={line.seq_no}>
      <div className="flex w-[32px] min-w-[32px] select-none justify-end text-gray-400">{line.seq_no}</div>
      <div className="flex items-center gap-x-1">
        <div className="flex h-full w-3 flex-col pt-1">{children}</div>
        <span className={`whitespace-pre-wrap ${line.kind === LogKind.Error && 'text-amaranth'}`}>{line.text}</span>
      </div>
    </div>
  );
}
