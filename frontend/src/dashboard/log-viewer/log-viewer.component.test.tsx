import React from 'react';
import { render, screen } from '@testing-library/react';
import { LogViewer } from './log-viewer.component';
import { IJobGraph } from '../../interfaces/job-graph.interface';
import * as logsHook from '../../hooks/logs/logs.hook';
import { mockJobGraph } from '../../mocks/models/jobGraph.mock';
import { mockUseLogDescriptor } from '../../mocks/hooks/useLogDescriptor.mock';

interface RenderOptions {
  indirectedJobGraph?: IJobGraph;
  jobGraph?: IJobGraph;
}

const defaultRenderOptions: RenderOptions = {
  jobGraph: mockJobGraph()
};

describe('LogViewer', () => {
  const renderLogViewer = (renderOptions?: RenderOptions): void => {
    mockUseLogDescriptor();

    const { indirectedJobGraph, jobGraph } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(<LogViewer indirectedJobGraph={indirectedJobGraph} jobGraph={jobGraph!} />);
  };

  it('should render an overview log group', () => {
    jest.spyOn(logsHook, 'useLogs').mockImplementation(() => {
      return { isLoadingLogs: false, logs: [] };
    });

    renderLogViewer();

    expect(screen.getByText('overview')).toBeInTheDocument();
  });

  it('should render a separate overview log group for an indirected job', () => {
    jest.spyOn(logsHook, 'useLogs').mockImplementation(() => {
      return { isLoadingLogs: false, logs: [] };
    });

    renderLogViewer({ indirectedJobGraph: mockJobGraph() });

    expect(screen.getByText('overview... skipped')).toBeInTheDocument();
    expect(screen.getByText('overview')).toBeInTheDocument();
  });
});
