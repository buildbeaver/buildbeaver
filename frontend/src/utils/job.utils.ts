import { IJob } from '../interfaces/job.interface';
import { IJobDependency } from '../interfaces/job-dependency.interface';

export function jobFQN(job: IJob): string {
  if (job.workflow) {
    return job.workflow + '.' + job.name;
  } else {
    return job.name;
  }
}

export function jobDependencyFQN(dep: IJobDependency): string {
  if (dep.workflow) {
    return dep.workflow + '.' + dep.job_name;
  } else {
    return dep.job_name;
  }
}
