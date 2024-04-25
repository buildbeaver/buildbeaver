/**
 * Used as effectively a cancellation token for searching.
 */
export class CancellableSearch {
  private isCancelledCore = false;
  private readonly queryCore: string;

  get isCancelled(): boolean {
    return this.isCancelledCore;
  }

  get query(): string {
    return this.queryCore;
  }

  constructor(query: string) {
    this.queryCore = query;
  }

  cancel(): void {
    this.isCancelledCore = true;
  }
}
