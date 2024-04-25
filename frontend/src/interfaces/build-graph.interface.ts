import { IBuild } from './build.interface';
import { IJobGraph } from './job-graph.interface';
import { IRepo } from './repo.interface';
import { ICommit } from './commit.interface';

export interface IBuildGraph {
  build: IBuild;
  commit: ICommit;
  jobs?: IJobGraph[];
  repo: IRepo;
  url: string;
}
