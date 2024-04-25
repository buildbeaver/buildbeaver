import { Edge, Node } from 'reactflow';
import { Status } from '../../../enums/status.enum';
import { INodeData } from '../interfaces/node-data.interface';
import { getColourForStatus } from '../../../utils/build-status.utils';
import { IJob } from '../../../interfaces/job.interface';
import { ITimings } from '../../../interfaces/timings.interface';
import { IJobGraph } from '../../../interfaces/job-graph.interface';
import { jobFQN } from '../../../utils/job.utils';

export abstract class BaseNode {
  abstract get height(): number;
  abstract get key(): string;
  abstract get width(): number;

  protected abstract get dependencyKeys(): string[];
  protected abstract get className(): string;
  protected abstract get edgeZIndex(): number;
  protected abstract get label(): string;
  protected abstract get parentKey(): string | undefined;
  protected abstract get runsOn(): string[] | undefined;
  protected abstract get style(): object;
  protected abstract get timings(): ITimings;
  protected abstract get type(): string;

  protected readonly jobCore: IJobGraph;

  private isDependency = false;
  private readonly dependencies: BaseNode[] = [];
  private readonly status: Status;

  get directDependencyKeys(): string[] {
    return this.directDependencies.map((dependency) => dependency.key);
  }

  get hasDependencies(): boolean {
    return this.dependencyKeys.length > 0;
  }

  protected get baseClassName(): string {
    return `border-2 border-${getColourForStatus(this.status)}`;
  }

  protected get baseStyle(): object {
    return {
      borderRadius: '0.25rem',
      height: this.height,
      width: this.width
    };
  }

  private get directDependencies(): BaseNode[] {
    const excludedKeys = this.dependencies.flatMap((dependency) => dependency.dependencyKeys);

    return this.dependencies.filter((dependency) => !excludedKeys.includes(dependency.key));
  }

  private get hasAnimatedEdge(): boolean {
    return this.directDependencies.some((flowNode) => flowNode.isRunning);
  }

  private get hasDirectDependency(): boolean {
    return this.directDependencyKeys.length > 0;
  }

  private get isRunning(): boolean {
    return this.status === Status.Queued || this.status === Status.Running;
  }

  protected get job(): IJob {
    return this.jobCore.job;
  }

  protected constructor(status: Status, job: IJobGraph) {
    this.status = status;
    this.jobCore = job;
  }

  addDependency(dependency: BaseNode): void {
    dependency.flagAsDependency();

    this.dependencies.push(dependency);
  }

  buildEdges(): Edge[] {
    return this.directDependencyKeys.map((dependencyKey) => {
      return {
        id: `${dependencyKey}-${this.key}`,
        source: dependencyKey,
        target: this.key,
        animated: this.hasAnimatedEdge,
        zIndex: this.edgeZIndex
      };
    });
  }

  buildNode(xPosition: number, yPosition: number): Node<INodeData> {
    return {
      id: this.key,
      className: this.className,
      data: {
        hideSourceHandle: !this.isDependency,
        hideTargetHandle: !this.hasDirectDependency,
        jobName: jobFQN(this.jobCore.job),
        label: this.label,
        status: this.status,
        runsOn: this.runsOn,
        timings: this.timings
      },
      parentNode: this.parentKey,
      position: {
        x: xPosition,
        y: yPosition
      },
      type: this.type,
      style: this.style
    };
  }

  flagAsDependency(): void {
    this.isDependency = true;
  }

  populateDependencies(nodeMap: { [key: string]: BaseNode }): void {
    if (this.hasDependencies) {
      for (const dependencyKey of this.dependencyKeys) {
        const subDependency = nodeMap[dependencyKey];
        // Ignore dependencies which don't refer to any existing node; these may be deferred dependencies
        if (subDependency) {
          this.addDependency(subDependency);
        }
      }
    }
  }
}
