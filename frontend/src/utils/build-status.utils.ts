import { Status } from '../enums/status.enum';

export function getColourForStatus(status: Status): string {
  let colour;

  switch (status) {
    case Status.Succeeded:
      colour = 'mountainMeadow';
      break;
    case Status.Failed:
      colour = 'amaranth';
      break;
    case Status.SkippedJob:
    case Status.SkippedStep:
      colour = 'curiousBlue';
      break;
    default:
      colour = 'athens';
  }

  return colour;
}

export function isFinished(status: Status): boolean {
  return [Status.Canceled, Status.Failed, Status.Succeeded, Status.SkippedJob, Status.SkippedStep].includes(status);
}
