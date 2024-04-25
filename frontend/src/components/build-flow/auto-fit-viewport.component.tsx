import React, { useEffect, useState } from 'react';
import { useReactFlow, Viewport } from 'reactflow';
import { isEqual } from 'lodash';
import { IJobGraph } from '../../interfaces/job-graph.interface';

interface Props {
  jobGraphs: IJobGraph[];
}

/**
 * Automatically runs fitView() on the DAG to recenter on all nodes as new jobs are added to a dynamic build. If the
 * user pans or zooms the viewport at any time this behaviour will stop.
 */
export function AutoFitViewport(props: Props): JSX.Element {
  const { jobGraphs } = props;
  const reactFlowInstance = useReactFlow();
  const [autoFittedViewport, setAutoFittedViewport] = useState<Viewport>({ x: 0, y: 0, zoom: 1 });

  useEffect(() => {
    const viewport = reactFlowInstance.getViewport();
    const isViewportUnchanged = isEqual(viewport, autoFittedViewport);

    if (isViewportUnchanged) {
      reactFlowInstance.fitView({ includeHiddenNodes: true });
      setAutoFittedViewport(reactFlowInstance.getViewport());
    }
  }, [jobGraphs.length]);

  useEffect(() => {
    // React flow reports viewport initialization earlier than we would like. We need to wrap this call to
    // setAutoFittedViewport() in setTimeout() so that we get the initialized viewport x and y position instead of
    // x: 0 and y: 0
    setTimeout(() => {
      setAutoFittedViewport(reactFlowInstance.getViewport());
    });
  }, [reactFlowInstance.viewportInitialized]);

  return <React.Fragment></React.Fragment>;
}
