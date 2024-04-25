export class UpdateToken {
  readonly id: string;
  private isUpdatingCore = true;

  get isUpdating(): boolean {
    return this.isUpdatingCore;
  }

  constructor(id: string) {
    this.id = id;
  }

  end(): void {
    this.isUpdatingCore = false;
  }
}
