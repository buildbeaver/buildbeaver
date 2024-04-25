import { ITimings } from './timings.interface';
import { Status } from '../enums/status.enum';
import { IStepDependency } from './step-dependency.interface';
import { IEnvironment } from './environment.interface';

export interface IStep {
  commands: string[];
  created_at: string;
  depends?: IStepDependency[];
  description: string;
  docker_authentication?: string;
  environment?: IEnvironment[];
  error?: string;
  etag: string;
  fingerprint?: string;
  fingerprint_commands?: string[];
  fingerprint_hash_type?: string;
  id: string;
  image?: string;
  indirect_to_step_id?: string;
  log_descriptor_id?: string;
  log_descriptor_url: string;
  name: string;
  pull?: string;
  repo_id: string;
  runner_id?: string;
  runner_url?: string;
  job_id: string;
  status: Status;
  timings: ITimings;
  updated_at: string;
  url: string;
}
