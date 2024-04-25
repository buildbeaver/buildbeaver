import React from 'react';
import { FaCheckCircle, FaClock, FaDirections, FaQuestionCircle, FaStopwatch, FaTimesCircle } from 'react-icons/fa';
import { BiLoader } from 'react-icons/bi';
import './build-status-indicator.component.scss';
import { Status } from '../../enums/status.enum';

interface Props {
  size?: number;
  status: Status;
}

/**
 * Indicates the status of a single build element. This could be a build or a job or a step.
 */
export function BuildStatusIndicator(props: Props): JSX.Element {
  const size = props.size ?? 22;

  let icon;

  switch (props.status) {
    case Status.Canceled:
      icon = <FaTimesCircle className="text-amaranth" size={size} title="Cancelled" />;
      break;
    case Status.Failed:
      icon = <FaTimesCircle className="text-amaranth" size={size} title="Failed" />;
      break;
    case Status.Succeeded:
      icon = <FaCheckCircle className="text-mountainMeadow" size={size} title="Succeeded" />;
      break;
    case Status.Queued:
      icon = <FaClock className="text-tundora" size={size} title="Queued" />;
      break;
    case Status.Running:
      icon = <BiLoader className="running text-mountainMeadow" size={size} title="Running" />;
      break;
    case Status.SkippedJob:
      icon = (
        <div className="relative" title="Skipped">
          <div className="absolute h-full w-full rounded-full border-[3px] border-curiousBlue"></div>
          <FaDirections className="text-curiousBlue" size={size} />
        </div>
      );
      break;
    case Status.SkippedStep:
      icon = <FaCheckCircle className="text-curiousBlue" size={size} title="Skipped" />;
      break;
    case Status.Submitted:
      icon = <FaStopwatch className="text-goldenBell" size={size} title="Submitted" />;
      break;
    default:
      icon = <FaQuestionCircle className="text-tundora" size={size} title="Unknown" />;
  }

  return icon;
}
