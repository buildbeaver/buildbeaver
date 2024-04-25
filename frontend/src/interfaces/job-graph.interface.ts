import { IStep } from './step.interface';
import { IJob } from './job.interface';

export interface IJobGraph {
  job: IJob;
  steps: IStep[];
  url: string;
}
