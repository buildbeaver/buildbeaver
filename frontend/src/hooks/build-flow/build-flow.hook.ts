import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { useEffect, useState } from 'react';
import ELK from 'elkjs/lib/elk-api';
import { Node } from 'reactflow';
import { makeBuildFlow, makeGraphData } from '../../components/build-flow/functions/build-flow.functions';
import { IBuildFlow } from '../../components/build-flow/interfaces/build-flow.interface';
import { isSkeletonErroredBuild } from '../../utils/build.utils';

export interface IUseBuildFlow {
  buildFlow?: IBuildFlow;
  buildFlowDrawing: boolean;
  buildFlowError?: Error;
}

export function useBuildFlow(buildGraph: IBuildGraph): IUseBuildFlow {
  const [buildFlow, setBuildFlow] = useState<IBuildFlow>();
  const [buildFlowDrawing, setBuildFlowDrawing] = useState(true);
  const [buildFlowError, setBuildFlowError] = useState<Error>();

  const createErrorNode = (): Node => {
    return {
      id: 'error',
      data: {
        message: 'Failed to render build graph'
      },
      position: {
        x: 0,
        y: 0
      },
      type: 'error'
    };
  };

  useEffect(() => {
    if (isSkeletonErroredBuild(buildGraph)) {
      setBuildFlow({ edges: [], nodes: [] });
      setBuildFlowDrawing(false);

      return;
    }

    try {
      const elk = new ELK({
        workerUrl: `${process.env.PUBLIC_URL}/scripts/elk-worker.min.js`
      });
      const graphData = makeGraphData(buildGraph.jobs!);
      const elkNode = {
        id: 'root',
        // https://www.eclipse.org/elk/reference/options.html
        layoutOptions: {
          'elk.algorithm': 'layered',
          'elk.layered.spacing.nodeNodeBetweenLayers': '50'
        },
        children: graphData.elkNodes,
        edges: graphData.elkEdges
      };

      elk
        .layout(elkNode)
        .then((elkGraph) => {
          setBuildFlow(makeBuildFlow(elkGraph, graphData));
        })
        .catch((error) => {
          setBuildFlowError(error);
        })
        .finally(() => {
          setBuildFlowDrawing(false);
        });
    } catch (error) {
      setBuildFlow({ edges: [], nodes: [createErrorNode()] });
      setBuildFlowDrawing(false);
    }
  }, [buildGraph]);

  return { buildFlow, buildFlowDrawing, buildFlowError };
}
