import { BaseNode } from './base-node.model';
import { BuildFlowUtils } from '../utils/build-flow.utils';
import { BuildFlowConstants } from '../constants/build-flow.constants';
import { ITimings } from '../../../interfaces/timings.interface';
import { IJobGraph } from '../../../interfaces/job-graph.interface';
import { StepNode } from './step-node.model';
import { jobDependencyFQN, jobFQN } from '../../../utils/job.utils';

export class JobNode extends BaseNode {
  private readonly stepNodesCore: StepNode[];

  get height(): number {
    return BuildFlowUtils.calculateStepYPosition(this.jobCore.steps.length);
  }

  get key(): string {
    return jobFQN(this.jobCore.job);
  }

  get stepNodes(): StepNode[] {
    return this.stepNodesCore;
  }

  get width(): number {
    return BuildFlowConstants.STEP.WIDTH + BuildFlowConstants.STEP.MARGIN * 2;
  }

  protected get className(): string {
    return `${this.baseClassName} bg-alabaster`;
  }

  protected get dependencyKeys(): string[] {
    return this.jobCore.job.depends?.map((dependency) => jobDependencyFQN(dependency)) ?? [];
  }

  protected get edgeZIndex(): number {
    return 0;
  }

  protected get label(): string {
    return jobFQN(this.jobCore.job);
  }

  protected get parentKey(): string | undefined {
    return undefined;
  }

  protected get runsOn(): string[] | undefined {
    return this.jobCore.job.runs_on;
  }

  protected get style(): object {
    return this.baseStyle;
  }

  protected get timings(): ITimings {
    return this.jobCore.job.timings;
  }

  protected get type(): string {
    return 'job';
  }

  constructor(jGraph: IJobGraph, stepNodes: StepNode[]) {
    super(jGraph.job.status, jGraph);
    this.stepNodesCore = stepNodes;
  }
}
