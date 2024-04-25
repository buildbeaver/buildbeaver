import { DateTime } from 'luxon';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { ITimings } from '../../interfaces/timings.interface';
import { IJobGraph } from '../../interfaces/job-graph.interface';

const defaultBuildGraph = {
  build: {
    id: 'test-build',
    name: '4',
    created_at: DateTime.now().plus({ day: -1, minute: -1 }).toISO(),
    ref: 'refs/heads/main',
    status: 'succeeded'
  },
  repo: {
    name: 'billys-playground',
    legal_entity_id: 'legal-entity:1527916f-6d25-49b8-b159-14f053eb7c7g'
  },
  commit: {
    sha: '6bdb713f07928245a862b5e2bd3adc1c3c3c7346',
    message: 'This is a test commit',
    author_name: 'Billy',
    author_email: 'billy@buildbeaver.com'
  }
} as unknown as IBuildGraph;

interface Options {
  jobs?: IJobGraph[];
  timings?: ITimings;
}

const defaultOptions: Options = {
  timings: {
    queued_at: '2022-08-29T02:13:07.890265Z',
    submitted_at: '2022-08-29T02:20:15.594971Z',
    running_at: '2022-08-29T02:20:15.711825Z',
    finished_at: '2022-08-29T02:22:53.859466Z',
    canceled_at: undefined
  }
};

export const mockBuildGraph = (options?: Options): IBuildGraph => {
  const { jobs, timings } = {
    ...defaultOptions,
    ...options
  };

  return {
    ...defaultBuildGraph,
    build: {
      ...defaultBuildGraph.build,
      timings: {
        ...defaultBuildGraph.build.timings,
        ...timings
      }
    },
    jobs
  };
};
