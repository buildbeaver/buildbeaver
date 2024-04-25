import { IJobGraph } from '../../../interfaces/job-graph.interface';
import { IStep } from '../../../interfaces/step.interface';
import { BuildFlowConstants } from '../constants/build-flow.constants';
import { BaseNode } from './base-node.model';
import { ITimings } from '../../../interfaces/timings.interface';
import { jobFQN } from '../../../utils/job.utils';

export class StepNode extends BaseNode {
  private readonly step: IStep;

  get height(): number {
    return BuildFlowConstants.STEP.HEIGHT;
  }

  get key(): string {
    return `${jobFQN(this.jobCore.job)}:${this.step.name}`;
  }

  get width(): number {
    return BuildFlowConstants.STEP.WIDTH;
  }

  protected get className(): string {
    return `${this.baseClassName} bg-white cursor-pointer`;
  }

  protected get dependencyKeys(): string[] {
    return this.step.depends?.map((dependency) => `${jobFQN(this.jobCore.job)}:${dependency.step_name}`) ?? [];
  }

  protected get edgeZIndex(): number {
    return 2;
  }

  protected get label(): string {
    return this.step.name;
  }

  protected get parentKey(): string | undefined {
    return jobFQN(this.jobCore.job);
  }

  protected get runsOn(): string[] | undefined {
    return undefined;
  }

  protected get style(): object {
    return {
      ...this.baseStyle,
      zIndex: 1
    };
  }

  protected get timings(): ITimings {
    return this.step.timings;
  }

  protected get type(): string {
    return 'step';
  }

  constructor(job: IJobGraph, step: IStep) {
    super(step.status, job);
    this.step = step;
  }
}
