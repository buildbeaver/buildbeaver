import { IArtifactDependency } from './artifact-dependency.interface';

export interface IJobDependency {
  artifact_dependencies?: IArtifactDependency[];
  job_name: string;
  workflow: string;
}
