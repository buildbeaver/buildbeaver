import * as buildFlowHook from '../../hooks/build-flow/build-flow.hook';
import { IJobGraph } from '../../interfaces/job-graph.interface';
import { makeBuildFlow, makeGraphData } from '../../components/build-flow/functions/build-flow.functions';
import { ElkNode } from 'elkjs/lib/elk.bundled';
import { jobFQN } from '../../utils/job.utils';

export function mockUseBuildFlow(jobGraphs: IJobGraph[]): void {
  jest.spyOn(buildFlowHook, 'useBuildFlow').mockImplementation(() => {
    const graphData = makeGraphData(jobGraphs);
    const mockElkGraph = {
      children: jobGraphs.map((jobGraph) => {
        return {
          id: jobFQN(jobGraph.job),
          x: 0,
          y: 0
        };
      })
    } as ElkNode;
    const mockBuildFlow = makeBuildFlow(mockElkGraph, graphData);

    return {
      buildFlow: mockBuildFlow,
      buildFlowDrawing: false
    };
  });
}
