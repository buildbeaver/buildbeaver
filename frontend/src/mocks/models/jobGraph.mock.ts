import { IJobGraph } from '../../interfaces/job-graph.interface';
import { mockJob } from './job.mock';

export const mockJobGraph = (): IJobGraph => {
  return {
    job: mockJob(),
    steps: [],
    url: ''
  };
};
