import { PollingConnectionStatus } from '../enums/polling-connection-status.enum';

/**
 * Indicates the polling connection status of a single instance of the usePolling hook.
 */
export class PollingConnection {
  private statusCore = PollingConnectionStatus.Retrying;

  get status(): PollingConnectionStatus {
    return this.statusCore;
  }

  abandon(): void {
    this.statusCore = PollingConnectionStatus.Abandoned;
  }

  restore(): void {
    this.statusCore = PollingConnectionStatus.Polling;
  }
}
