import { ElkExtendedEdge, ElkNode } from 'elkjs/lib/elk.bundled';
import { IJobGraph } from '../../../interfaces/job-graph.interface';
import { IGraphData } from '../interfaces/graph-data.interface';
import { JobNode } from '../models/job-node.model';
import { BaseNode } from '../models/base-node.model';
import { StepNode } from '../models/step-node.model';
import { Edge, Node } from 'reactflow';
import { BuildFlowConstants } from '../constants/build-flow.constants';
import { BuildFlowUtils } from '../utils/build-flow.utils';
import { IBuildFlow } from '../interfaces/build-flow.interface';

/**
 * Uses the calculated X and Y positions of the job nodes in the elk graph to finalise laying out the build flow.
 */
export function makeBuildFlow(elkGraph: ElkNode, graphData: IGraphData): IBuildFlow {
  let edges: Edge[] = [];
  let nodes: Node[] = [];

  elkGraph.children?.forEach((elkNode) => {
    const jobNode = graphData.jobNodes?.find((job) => job.key === elkNode.id)!;

    edges = [...edges, ...jobNode.buildEdges()];
    nodes = [...nodes, jobNode.buildNode(elkNode.x!, elkNode.y!)];

    jobNode.stepNodes.forEach((stepNode, index) => {
      edges = [...edges, ...stepNode.buildEdges()];
      nodes = [...nodes, stepNode.buildNode(BuildFlowConstants.STEP.MARGIN, BuildFlowUtils.calculateStepYPosition(index))];
    });
  });

  return { edges, nodes };
}

/**
 * Transforms a builds jobs into a format that Elk can use to lay out the job node X and Y positions.
 */
export function makeGraphData(jobs: IJobGraph[]): IGraphData {
  const jobNodes = jobs.map((job) => {
    const stepNodes = processSteps(job);

    return new JobNode(job, stepNodes);
  });

  populateNodeDependencies(jobNodes);

  return {
    elkEdges: makeElkEdges(jobNodes),
    elkNodes: makeElkNodes(jobNodes),
    jobNodes
  };
}

/**
 * We need to provide edges to Elk so that it can lay out the job nodes with overlapping edges.
 */
function makeElkEdges(jobNodes: JobNode[]): ElkExtendedEdge[] {
  return jobNodes.flatMap((jobNode) => {
    return jobNode.directDependencyKeys.map((key) => {
      return {
        id: `${key}-${jobNode.key}`,
        sources: [key],
        targets: [jobNode.key]
      };
    });
  });
}

/**
 * Transforms out jobs to Elk nodes with widths and heights precalculated (based on number of steps)
 */
function makeElkNodes(jobNodes: JobNode[]): ElkNode[] {
  return jobNodes.map((jobNode) => {
    return {
      id: jobNode.key,
      height: jobNode.height,
      width: jobNode.width
    };
  });
}

function populateNodeDependencies(nodes: BaseNode[]): void {
  const nodeMap: { [key: string]: BaseNode } = nodes.reduce((map, node) => ({ ...map, [node.key]: node }), {});

  for (const node of nodes) {
    node.populateDependencies(nodeMap);
  }
}

function processSteps(job: IJobGraph): StepNode[] {
  const stepNodes = job.steps.map((step) => new StepNode(job, step));

  populateNodeDependencies(stepNodes);

  return stepNodes;
}
