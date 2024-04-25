import * as buildHook from '../../hooks/build/build.hook';
import { IBuildGraph } from '../../interfaces/build-graph.interface';

export function mockUseBuild(buildGraph: IBuildGraph): void {
  jest.spyOn(buildHook, 'useBuild').mockImplementation(() => {
    return {
      buildGraph: buildGraph
    };
  });
}
