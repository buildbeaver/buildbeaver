import { IJobGraph } from '../../interfaces/job-graph.interface';
import { SortDecorator } from './sort-decorator';
import { jobDependencyFQN, jobFQN } from '../job.utils';

/**
 * Provides extra behaviour to jobs when sorting a build.
 */
export class JobSortDecorator extends SortDecorator<IJobGraph> {
  get dependencyKeys(): string[] {
    return this.node.job.depends?.map((dependency) => jobDependencyFQN(dependency)) ?? [];
  }

  get name(): string {
    return jobFQN(this.node.job);
  }
}
