import React, { useEffect, useState } from 'react';
import { LogLine } from '../log-line/log-line.component';
import { ILogBlock } from '../../interfaces/log-block.interface';
import { LogKind } from '../../enums/log-kind.enum';
import { IoTriangleSharp } from 'react-icons/io5';

interface Props {
  logBlock: ILogBlock;
}

export function LogBlock(props: Props): JSX.Element {
  const { logBlock } = props;
  const { block, lines, shouldExpand } = logBlock;
  const [isExpanded, setIsExpanded] = useState(false);

  useEffect(() => {
    setIsExpanded(shouldExpand);
  }, [shouldExpand]);

  const hasLines = (): boolean => {
    const hasMultipleLines = lines.length > 1;
    const hasSingleLine = !hasMultipleLines && lines.length === 1 && lines[0].kind !== LogKind.LogEnd;

    return hasMultipleLines || hasSingleLine;
  };

  const blockClicked = (): void => {
    setIsExpanded(!isExpanded);
  };

  if (hasLines()) {
    return (
      <div className="flex flex-col">
        <div className="cursor-pointer" key={block.seq_no} onClick={blockClicked}>
          <LogLine line={block}>
            <IoTriangleSharp className={isExpanded ? 'rotate-180' : 'rotate-90'} size={10} />
          </LogLine>
        </div>
        {isExpanded && lines.map((line, index) => <LogLine line={line} key={index} />)}
      </div>
    );
  }

  return <LogLine key={block.seq_no} line={block} />;
}
