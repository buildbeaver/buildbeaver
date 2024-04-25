import { buildDuration, createdAt, patchSkippedStatuses } from './build.utils';
import { ITimings } from '../interfaces/timings.interface';
import { DateTime, Duration } from 'luxon';
import { IBuildGraph } from '../interfaces/build-graph.interface';
import { Status } from '../enums/status.enum';
import { IJob } from '../interfaces/job.interface';

interface timing {
  finished?: Duration;
  queued?: Duration;
  running?: Duration;
  submitted?: Duration;
}

const createTimings = (timing: timing): ITimings => {
  const now = DateTime.utc();

  return {
    finished_at: now.plus(timing.finished || 0).toISO(),
    queued_at: now.plus(timing.queued || 0).toISO(),
    running_at: now.plus(timing.running || 0).toISO(),
    submitted_at: now.plus(timing.submitted || 0).toISO()
  };
};

describe('build.utils', () => {
  describe('buildDuration', () => {
    it('should return minutes and seconds for build under 60 minutes', () => {
      const timings = createTimings({ finished: Duration.fromObject({ minutes: 59, milliseconds: 123 }) });

      expect(buildDuration(timings)).toBe('59m 00s');
    });

    it('should return hours and minutes for build over 60 minutes', () => {
      const timings = createTimings({ finished: Duration.fromObject({ minutes: 61 }) });

      expect(buildDuration(timings)).toBe('01h 01m');
    });

    it('should return hours and minutes for build at 60 minutes with seconds', () => {
      const timings = createTimings({ finished: Duration.fromObject({ minutes: 60, seconds: 10 }) });

      expect(buildDuration(timings)).toBe('01h 00m');
    });

    it('should return hours and minutes for build over 60 minutes with seconds', () => {
      const timings = createTimings({ finished: Duration.fromObject({ minutes: 61, seconds: 10 }) });

      expect(buildDuration(timings)).toBe('01h 01m');
    });

    it('should return hours for build over 99 hours', () => {
      const timings = createTimings({ finished: Duration.fromObject({ hours: 100 }) });

      expect(buildDuration(timings)).toBe('99h+');
    });
  });

  describe('createdAt', () => {
    it('should return Just now for dates within 5 seconds', () => {
      const expected = 'Just now';
      let utcDate = DateTime.utc();

      expect(createdAt(utcDate.toISO())).toBe(expected);

      utcDate = utcDate.minus({ second: 4 });

      expect(createdAt(utcDate.toISO())).toBe(expected);

      utcDate = utcDate.minus({ second: 5 });

      expect(createdAt(utcDate.toISO())).not.toBe(expected);
    });
  });

  describe('patchSkippedStatuses', () => {
    it('should handle an empty build graph', () => {
      const buildGraph = {} as IBuildGraph;

      expect(patchSkippedStatuses(buildGraph)).toEqual(buildGraph);
    });

    it('should patch skipped statuses onto jobs and steps', () => {
      const buildGraph = {
        jobs: [
          {
            job: {
              name: 'job 1',
              status: Status.Succeeded,
              indirect_to_job_id: '123'
            } as IJob,
            steps: [
              {
                name: 'step 1',
                status: Status.Succeeded
              },
              {
                name: 'step 2',
                status: Status.Succeeded
              }
            ]
          },
          {
            job: {
              name: 'job 2',
              status: Status.Succeeded
            } as IJob,
            steps: [
              {
                name: 'step 3',
                status: Status.Succeeded
              }
            ]
          }
        ]
      } as IBuildGraph;

      expect(patchSkippedStatuses(buildGraph)).toEqual({
        jobs: [
          {
            job: {
              name: 'job 1',
              status: Status.SkippedJob,
              indirect_to_job_id: '123'
            },
            steps: [
              {
                name: 'step 1',
                status: Status.SkippedStep
              },
              {
                name: 'step 2',
                status: Status.SkippedStep
              }
            ]
          },
          {
            job: {
              name: 'job 2',
              status: Status.Succeeded
            },
            steps: [
              {
                name: 'step 3',
                status: Status.Succeeded
              }
            ]
          }
        ]
      });
    });
  });
});
