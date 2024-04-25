import { IBuildGraph } from './build-graph.interface';

export interface IBuildsSummary {
  completed?: IBuildGraph[];
  running?: IBuildGraph[];
  upcoming?: IBuildGraph[];
}
