/**
 * Wraps jobs and steps so that we can process them using the same logic even though they store their dependency
 * keys in different data structures.
 */
export abstract class SortDecorator<TNode> {
  /**
   * The names of jobs or steps that the underlying job or step is dependent on.
   */
  abstract get dependencyKeys(): string[];

  /**
   * The job or step name used as a key when sorting builds.
   */
  abstract get name(): string;

  /**
   * So we can read back the underling job or step once sorting is complete.
   */
  get node(): TNode {
    return this.nodeCore;
  }

  private readonly nodeCore: TNode;

  constructor(node: TNode) {
    this.nodeCore = node;
  }
}
