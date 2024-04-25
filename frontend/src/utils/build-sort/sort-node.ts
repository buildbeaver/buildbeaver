/**
 * Tracks a job or step while sorting a build.
 */
export class SortNode {
  private dependenciesCore: string[] = [];
  private dependentsCore: string[] = [];

  /**
   * Keys of nodes that should be rendered before this node.
   */
  get dependencies(): string[] {
    return this.dependenciesCore;
  }

  /**
   * Keys of nodes that should be rendered after this node.
   */
  get dependents(): string[] {
    return this.dependentsCore;
  }

  /**
   * Job or step name.
   */
  key: string;

  get hasNoDependencies(): boolean {
    return this.dependencies.length === 0;
  }

  get hasNoDependents(): boolean {
    return this.dependents.length === 0;
  }

  constructor(key: string) {
    this.key = key;
  }

  addDependency(dependencyKey: string): void {
    this.dependenciesCore = [...this.dependenciesCore, dependencyKey];
  }

  addDependent(dependentKey: string): void {
    this.dependentsCore = [...this.dependentsCore, dependentKey];
  }
}
