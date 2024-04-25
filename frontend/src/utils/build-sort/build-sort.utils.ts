import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IJobGraph } from '../../interfaces/job-graph.interface';
import { IStep } from '../../interfaces/step.interface';
import { SortNode } from './sort-node';
import { JobSortDecorator } from './job-sort-decorator';
import { SortDecorator } from './sort-decorator';
import { StepSortDecorator } from './step-sort-decorator';
import { isSkeletonErroredBuild } from '../build.utils';

/**
 * Sorts the jobs and steps for a build in a logical order based on the dependency tree formed by the nodes. This lets
 * us render the build tree and build chart in a way that visually makes sense, regardless of what order the jobs and
 * steps are served in.
 */
export function sortBuild(bGraph: IBuildGraph): IBuildGraph {
  // Ensure that we do not perform any job sorting actions if we have a skeleton build without jobs
  if (isSkeletonErroredBuild(bGraph)) {
    return { ...bGraph };
  }

  return {
    ...bGraph,
    jobs:
      bGraph.jobs &&
      sortJobs(
        bGraph.jobs.map((job) => {
          return {
            ...job,
            steps: sortSteps(job.steps)
          };
        })
      )
  };
}

/**
 * Sorts steps for a single job based on their dependency tree.
 */
export function sortJob(jobGraph: IJobGraph): IJobGraph {
  return new JobSortDecorator(jobGraph).node;
}

/**
 * Sorts jobs for a build based on their dependency tree.
 */
function sortJobs(jobs: IJobGraph[]): IJobGraph[] {
  let jobNodes = jobs.map((job) => new JobSortDecorator(job));
  let sortedJobNodes = sortBuildNodes<IJobGraph>(jobNodes);
  return sortedJobNodes.map((sortedJob) => sortedJob.node);
}

/**
 * Sorts steps for a build based on their dependency tree.
 */
function sortSteps(steps: IStep[]): IStep[] {
  let stepNodes = steps.map((step) => new StepSortDecorator(step));
  let sortedStepNodes = sortBuildNodes<IStep>(stepNodes);
  return sortedStepNodes.map((sortedStep) => sortedStep.node);
}

/**
 * Processes the given list of build nodes and transforms them into sort nodes where we can track both the dependencies
 * and dependents for each node. We need to know both so that we when finally sort we can slot each node in after its
 * dependencies and before its dependents.
 * A sort node will be created for each dependency, even if the node depended on doesn't exist yet (deferred
 * dependencies) so the returned map may contain more entries than the supplied buildNodes array.
 */
function buildSortNodeMap<TNode>(buildNodes: SortDecorator<TNode>[]): { [key: string]: SortNode } {
  const sortNodeMap: { [key: string]: SortNode } = {};

  for (const buildNode of buildNodes) {
    const sortNode = sortNodeMap[buildNode.name] ?? new SortNode(buildNode.name);

    for (const dependencyKey of buildNode.dependencyKeys) {
      // Note that all dependencies are added here, regardless of whether they were in the original list of nodes
      const dependencyNode = sortNodeMap[dependencyKey] ?? new SortNode(dependencyKey);

      dependencyNode.addDependent(buildNode.name);
      sortNode.addDependency(dependencyKey);

      sortNodeMap[dependencyNode.key] = dependencyNode;
    }

    sortNodeMap[buildNode.name] = sortNode;
  }

  return sortNodeMap;
}

/**
 * Converts the given list of build nodes to sort nodes and sorts them so that a node always comes after its dependencies
 * and before its dependents.
 */
function sortBuildNodes<TNode>(buildNodes: SortDecorator<TNode>[]): SortDecorator<TNode>[] {
  const sortNodeMap = buildSortNodeMap(buildNodes);
  const sortedKeys: string[] = [];

  let independentCount = 0;

  for (const [nodeKey, sortNode] of Object.entries(sortNodeMap)) {
    // No dependents or dependencies, slot this node in right at the start.
    if (sortNode.hasNoDependents && sortNode.hasNoDependencies) {
      sortedKeys.unshift(nodeKey);
      independentCount++;
      continue;
    }

    const precedingIndexes = sortNode.dependencies
      .map((dependencyKey) => sortedKeys.indexOf(dependencyKey))
      .filter((index) => index !== -1);

    if (precedingIndexes.some((index) => index > -1)) {
      // We processed one or more dependencies for this node already, slot this node in after them.
      const highestIndex = Math.max(...precedingIndexes);
      sortedKeys.splice(highestIndex + 1, 0, nodeKey);
      continue;
    }

    const followingIndexes = sortNode.dependents
      .map((dependencyKey) => sortedKeys.indexOf(dependencyKey))
      .filter((index) => index !== -1);

    if (followingIndexes.some((index) => index > -1)) {
      // We processed one or more dependants for this node already, slot this node in before them.
      const lowestIndex = Math.min(...followingIndexes);
      sortedKeys.splice(lowestIndex, 0, nodeKey);
      continue;
    }

    // No dependents or dependencies processed yet for this node, slot this node in after all independent nodes.
    sortedKeys.splice(independentCount, 0, nodeKey);
  }

  // Map the sorted keys back into nodes (e.g. jobs or steps).
  // Some sorted keys may be for 'deferred dependencies' which don't have a node yet, so only add results
  // for nodes which exist in the original buildNodes list.
  const result: SortDecorator<TNode>[] = [];
  sortedKeys.forEach((key) => {
    const node = buildNodes.find((node) => node.name === key);
    if (node) {
      result.push(node);
    }
  });

  return result;
}
