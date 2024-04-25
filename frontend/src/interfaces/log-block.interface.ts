import { IStructuredLog } from './structured-log.interface';

export interface ILogBlock {
  block: IStructuredLog;
  lines: IStructuredLog[];
  shouldExpand: boolean;
}
