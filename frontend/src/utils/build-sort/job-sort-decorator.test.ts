import { JobSortDecorator } from './job-sort-decorator';
import { IJobGraph } from '../../interfaces/job-graph.interface';

describe('job-sort-decorator', () => {
  it('should extract dependency keys from a job', () => {
    const jobGraph = {
      job: {
        name: 'js-test',
        depends: [
          {
            job_name: 'base',
            artifact_dependencies: undefined
          },
          {
            job_name: 'generate',
            artifact_dependencies: undefined
          }
        ]
      },
      steps: {}
    } as IJobGraph;

    const jobSortDecorator = new JobSortDecorator(jobGraph);

    expect(jobSortDecorator.name).toBe('js-test');
    expect(jobSortDecorator.dependencyKeys).toEqual(['base', 'generate']);
  });
});
