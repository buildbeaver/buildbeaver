import { IJob } from '../../interfaces/job.interface';
import { Status } from '../../enums/status.enum';

const defaultJob = {
  id: 'test-job',
  name: 'test-job',
  status: Status.Succeeded
} as IJob;

interface Options {
  status?: Status;
}

const defaultOptions: Options = {
  status: Status.Succeeded
};

export const mockJob = (options?: Options): IJob => {
  const { status } = {
    ...defaultOptions,
    ...options
  };

  return {
    ...defaultJob,
    status: status!,
    timings: {}
  };
};
