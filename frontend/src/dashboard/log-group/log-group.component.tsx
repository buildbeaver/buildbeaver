import { useLogs } from '../../hooks/logs/logs.hook';
import React, { useEffect, useState } from 'react';
import { IStructuredLog } from '../../interfaces/structured-log.interface';
import { LogKind } from '../../enums/log-kind.enum';
import { BuildStatusIndicator } from '../build-status-indicator/build-status-indicator.component';
import { LogBlock } from '../log-block/log-block.component';
import { ILogBlock } from '../../interfaces/log-block.interface';
import { Status } from '../../enums/status.enum';
import { IoChevronForwardSharp } from 'react-icons/io5';
import { Loading } from '../../components/loading/loading.component';
import { ITimings } from '../../interfaces/timings.interface';
import { Timer } from '../../components/timer/timer.component';
import { StructuredErrorMessage } from '../../components/structured-error-message/structured-error-message.component';
import { LogLine } from '../log-line/log-line.component';

interface Props {
  error?: string;
  forceCollapse: boolean;
  id: string;
  logDescriptorUrl: string;
  name: string;
  status: Status;
  timings: ITimings;
}

export function LogGroup(props: Props): JSX.Element {
  const { error, forceCollapse, id, logDescriptorUrl, name, status, timings } = props;
  const { logs, logsError, isLoadingLogs } = useLogs(logDescriptorUrl);
  const [isExpanded, setIsExpanded] = useState(false);

  const hasLogs = !!logs && logs.length > 1;
  const shouldExpand = !forceCollapse && hasLogs && logs[logs.length - 1].kind !== LogKind.LogEnd;

  useEffect(() => {
    setIsExpanded(shouldExpand);
  }, [shouldExpand]);

  useEffect(() => {
    if (forceCollapse) {
      setIsExpanded(false);
    }
  }, [forceCollapse]);

  const groupClicked = (): void => {
    setIsExpanded(!isExpanded);
  };

  const newBlock = (log: IStructuredLog): ILogBlock => {
    return {
      block: log,
      lines: [],
      shouldExpand: false
    };
  };

  const buildBlocks = (logs: IStructuredLog[]): JSX.Element[] => {
    const logBlocks: ILogBlock[] = [];
    let lastLogBlock: ILogBlock | undefined;

    for (const log of logs) {
      if (log.kind === LogKind.Block) {
        const block = newBlock(log);

        lastLogBlock = block;
        logBlocks.push(block);
      } else if (lastLogBlock && log.parent_block_name === lastLogBlock.block.name) {
        lastLogBlock.lines.push(log);
      } else {
        logBlocks.push(newBlock(log));
      }
    }

    if (logBlocks.length > 0) {
      logBlocks[logBlocks.length - 1].shouldExpand = status === Status.Running;
    }

    return logBlocks.map((block, index) => <LogBlock logBlock={block} key={index} />);
  };

  return (
    <div className="relative flex flex-col font-mono">
      <div className={`bg-gray-800 ${isExpanded && 'sticky top-[2.8rem] z-[1]'}`}>
        <div
          className={`flex w-full cursor-pointer justify-between gap-x-4 rounded-md p-1 hover:bg-athensTransparent ${
            isExpanded && 'bg-athensTransparent'
          }`}
          onClick={() => hasLogs && groupClicked()}
        >
          <div className="flex min-w-0 items-center gap-x-2">
            <div className="h-3 w-3">
              {hasLogs && <IoChevronForwardSharp className={`${isExpanded && 'rotate-90'}`} />}
              {isLoadingLogs && <Loading />}
            </div>
            <div>
              <BuildStatusIndicator status={status} size={16} />
            </div>
            <span className="... truncate" title={name}>
              {name}
            </span>
          </div>
          <div className="pr-2">
            <Timer className="text-gray-400" key={id} timings={timings} />
          </div>
        </div>
      </div>
      {logs && isExpanded && <div className="py-0.5">{buildBlocks(logs)}</div>}
      {logs && !hasLogs && error && <LogLine line={{ kind: LogKind.Error, text: error } as IStructuredLog} />}
      {logsError && (
        <span className="ml-12 font-sans text-amaranth">
          <StructuredErrorMessage error={logsError} fallback="Failed to fetch logs" />
        </span>
      )}
    </div>
  );
}
