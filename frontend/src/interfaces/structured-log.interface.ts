import { LogKind } from '../enums/log-kind.enum';

export interface IStructuredLog {
  client_timestamp: string;
  kind: LogKind;
  line_no: number;
  name: string;
  parent_block_name?: string;
  seq_no: number;
  server_timestamp: string;
  text: string;
}
