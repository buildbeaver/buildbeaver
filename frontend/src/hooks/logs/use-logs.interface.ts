import { IStructuredLog } from '../../interfaces/structured-log.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';

export interface IUseLogs {
  logs?: IStructuredLog[];
  logsError?: IStructuredError;
  isLoadingLogs: boolean;
}
