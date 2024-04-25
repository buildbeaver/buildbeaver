import React, { useContext } from 'react';
import { NavLink } from 'react-router-dom';
import { IoCalendarClearOutline } from 'react-icons/io5';
import { getColourForStatus } from '../../utils/build-status.utils';
import { IRunner } from '../../interfaces/runner.interface';
import { Status } from '../../enums/status.enum';
import { PlatformIndicator } from '../platform-indicator/platform-indicator.component';
import { SetupContext } from '../../contexts/setup/setup.context';
import { createdAt } from '../../utils/build.utils';

interface Props {
  isFirst: boolean;
  isLast: boolean;
  runner: IRunner;
}

/**
 * Constructs an individual Runner list item
 */
export function RunnerListItem(props: Props): JSX.Element {
  const { isInSetupContext } = useContext(SetupContext);
  const { isFirst, isLast, runner } = props;

  const backgroundColour = (): string => {
    return `bg-${getColourForStatus(runner.enabled ? Status.Succeeded : Status.Failed)}`; // TODO: Move to an actual status when we have them for Runners
  };

  const borderRadiusStyle = (): string => {
    if (isFirst && isLast) {
      return 'rounded-l-md';
    }

    if (isFirst) {
      return 'rounded-tl-md';
    }

    if (isLast) {
      return 'rounded-bl-md';
    }

    return '';
  };

  const content = (): JSX.Element => {
    return (
      <>
        <div className="flex min-w-0 grow">
          <div className={`w-[4px] min-w-[4px] ${backgroundColour()} ${borderRadiusStyle()}`}></div>
          <div className="flex w-[60%] min-w-[60%] gap-x-2 p-3">
            <div className="flex min-w-0 flex-col gap-y-1">
              <div className="... truncate" title={runner.name}>
                {runner.name}
              </div>
            </div>
          </div>
          <div className="flex w-[15%] min-w-[15%] gap-x-2 p-3">
            <div className="flex min-w-0 gap-x-1">
              <div>
                <PlatformIndicator runsOn={[runner.operating_system, runner.architecture]} />
              </div>
              <div className="... truncate" title={`Operating system: ${runner.operating_system}`}>
                <strong>{runner.operating_system}</strong>
              </div>
            </div>
          </div>
          <div className="flex w-[15%] min-w-[15%] gap-x-2 p-3">
            <div className="flex min-w-0 flex-col gap-y-1">
              <div className="... truncate" title={`Architecture: ${runner.architecture}`}>
                <strong>{runner.architecture}</strong>
              </div>
            </div>
          </div>
          <div className="flex w-[10%] min-w-[10%] gap-x-2 p-3">
            <div className="flex min-w-0 flex-col gap-y-1">
              <div className="... truncate" title={`Software version: ${runner.software_version}`}>
                <strong>{runner.software_version}</strong>
              </div>
            </div>
          </div>
        </div>
        <div className="flex w-[150px] min-w-[150px] flex-col justify-center gap-y-1 p-3 text-xs">
          <div className="flex items-center gap-x-2" title="Created">
            <IoCalendarClearOutline className="ml-[1px]" size={14} />
            {createdAt(runner.created_at)}
          </div>
        </div>
      </>
    );
  };

  const commonStyles = 'flex justify-between text-sm text-gray-600';

  if (isInSetupContext) {
    return <div className={commonStyles}>{content()}</div>;
  }

  return (
    <NavLink className={commonStyles} to={runner.name}>
      {content()}
    </NavLink>
  );
}
