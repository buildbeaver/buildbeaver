import React, { useCallback, useMemo } from 'react';
import { StepNode } from './nodes/step-node.component';
import { JobNode } from './nodes/job-node.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { INodeData } from './interfaces/node-data.interface';
import { ErrorNode } from './nodes/error-node.component';
import { useLocation, useNavigate } from 'react-router-dom';
import { replacePathParts } from '../../utils/path.utils';
import { FitViewOptions, ReactFlow, Node, Controls } from 'reactflow';
import { AutoFitViewport } from './auto-fit-viewport.component';
import { useBuildFlow } from '../../hooks/build-flow/build-flow.hook';
import { Error } from '../error/error.component';

// Importing CSS is mandatory in React Flow v11
import 'reactflow/dist/base.css';
import { SimpleContentLoader } from '../content-loaders/simple/simple-content-loader';

interface Props {
  bGraph: IBuildGraph;
}

export function BuildFlow(props: Props): JSX.Element {
  const { bGraph } = props;
  const navigate = useNavigate();
  const location = useLocation();
  const nodeTypes = useMemo(() => ({ error: ErrorNode, job: JobNode, step: StepNode }), []);
  const { buildFlow, buildFlowDrawing, buildFlowError } = useBuildFlow(bGraph);
  const fitViewOptions: FitViewOptions = {
    minZoom: 0
  };

  const onNodeClick = useCallback(
    (event: React.MouseEvent<Element, MouseEvent>, node: Node<INodeData>) => {
      // Navigate to the build for the clicked job.
      const path = replacePathParts(location.pathname, [{ positionFromEnd: 1, replacement: `log/${node.data.jobName}` }]);
      navigate(path);
    },
    [location.pathname, navigate]
  );

  if (buildFlowError) {
    return <Error errorMessage={buildFlowError.message} />;
  }

  if (buildFlowDrawing || !buildFlow) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={450} />;
  }

  const { edges, nodes } = buildFlow;

  return (
    <div className="grow">
      <ReactFlow
        className={`cursor-grab rounded-md border-2 bg-alabaster shadow-md`}
        nodeTypes={nodeTypes}
        nodes={nodes}
        edges={edges}
        fitView={true}
        fitViewOptions={fitViewOptions}
        minZoom={0.2}
        maxZoom={0.8}
        nodesDraggable={false}
        onNodeClick={onNodeClick}
      >
        <Controls position="top-right" showInteractive={false} />
        {bGraph.jobs && <AutoFitViewport jobGraphs={bGraph.jobs} />}
      </ReactFlow>
    </div>
  );
}
