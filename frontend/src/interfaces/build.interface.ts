import { ITimings } from './timings.interface';
import { Status } from '../enums/status.enum';

export interface IBuild {
  artifact_search_url: string;
  commit_id: string;
  created_at: string;
  error?: string;
  etag: string;
  id: string;
  log_descriptor_id?: string;
  log_descriptor_url: string;
  name: string;
  opts: {
    force?: boolean;
    nodes_to_run?: string;
  };
  ref: string;
  repo_id?: string;
  status: Status;
  timings: ITimings;
  updated_at: string;
  url: string;
}
