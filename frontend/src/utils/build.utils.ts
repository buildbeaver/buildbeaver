import { IBuildGraph } from '../interfaces/build-graph.interface';
import { Status } from '../enums/status.enum';
import { ITimings } from '../interfaces/timings.interface';
import { DateTime } from 'luxon';
import { getColourForStatus } from './build-status.utils';

/**
 * Build utility to get the background color based on the builds status.
 */
export function backgroundColour(status: Status): string {
  return `bg-${getColourForStatus(status)}`;
}

/**
 * Build utility to return a string for the duration a build has been running for.
 */
export function buildDuration(timings: ITimings): string {
  if (timings.running_at) {
    const endTime = timings.finished_at ?? timings.canceled_at ?? DateTime.utc().toISO();

    // Normalize the end time timing first
    const duration = DateTime.fromISO(endTime)
      .diff(DateTime.fromISO(timings.running_at), ['hours', 'minutes', 'seconds'])
      .normalize();

    if (duration.hours > 0) {
      if (duration.hours > 99) {
        return '99h+';
      }

      return duration.toFormat("hh'h' mm'm'");
    }

    return duration.toFormat("mm'm' ss's'");
  }

  if (timings.submitted_at) {
    return 'Submitted';
  }

  return 'Queued';
}

/**
 * Build utility to turn the createdAt string into a human-readable string.
 */
export function createdAt(createdAt: string): string | null {
  const date = DateTime.fromISO(createdAt);

  // If created at was within 5 seconds then display Just Now
  if (date.diffNow().as('seconds') > -5) {
    return 'Just now';
  }

  return date.toRelative();
}

/**
 * Build utility to check if we have a skeleton errored build that does not contain any jobs.
 */
export function isSkeletonErroredBuild(bGraph?: IBuildGraph): boolean {
  // Ensure that we do not perform any actions if we have a skeleton build that is failed and without jobs.
  return bGraph?.build.status === Status.Failed && !bGraph?.jobs;
}

/**
 * Updates the statuses of jobs and steps to mark them as skipped where an indirected job is detected. This is
 * temporary until the API can provide skipped statuses.
 */
export function patchSkippedStatuses(bGraph: IBuildGraph): IBuildGraph {
  if (!bGraph.jobs) {
    return bGraph;
  }

  return {
    ...bGraph,
    jobs: [
      ...bGraph.jobs.map((jGraph) => {
        if (!jGraph.job.indirect_to_job_id) {
          return jGraph;
        }

        return {
          ...jGraph,
          job: {
            ...jGraph.job,
            status: Status.SkippedJob
          },
          steps: [
            ...jGraph.steps.map((step) => {
              return {
                ...step,
                status: Status.SkippedStep
              };
            })
          ]
        };
      })
    ]
  };
}

/**
 * Git utility to trim a ref into a human-readable string for the front-end.
 */
export function trimRef(ref: string): string {
  if (ref.startsWith('refs/')) {
    ref = ref.slice(5);
  }

  if (ref.startsWith('heads/')) {
    ref = ref.slice(6);
  }

  return ref;
}

/**
 * Git utility to trim an SHA down to its first 12 characters.
 */
export function trimSha(sha: string): string {
  return sha.slice(0, 12);
}
