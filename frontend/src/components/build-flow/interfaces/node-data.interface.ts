import { Status } from '../../../enums/status.enum';
import { ITimings } from '../../../interfaces/timings.interface';

export interface INodeData {
  // Handles are always present but hidden until required - to support the addition of nodes during dynamic builds.
  hideSourceHandle: boolean;
  hideTargetHandle: boolean;
  jobName: string;
  label: string;
  runsOn?: string[];
  status: Status;
  timings: ITimings;
}
