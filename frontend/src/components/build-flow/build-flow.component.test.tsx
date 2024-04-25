import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { render, screen, waitFor } from '@testing-library/react';
import { BuildFlow } from './build-flow.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { Status } from '../../enums/status.enum';
import { mockUseBuildFlow } from '../../mocks/hooks/useBuildFlow.mock';

window.ResizeObserver =
  window.ResizeObserver ||
  jest.fn().mockImplementation(() => ({
    disconnect: jest.fn(),
    observe: jest.fn(),
    unobserve: jest.fn()
  }));

describe('BuildFlow', () => {
  const renderBuildFlow = (bGraph: IBuildGraph): void => {
    mockUseBuildFlow(bGraph.jobs!);
    render(
      <BrowserRouter>
        <BuildFlow bGraph={bGraph} />
      </BrowserRouter>
    );
  };

  it('should render a job', async () => {
    const bgGraph = {
      build: {
        status: Status.Succeeded,
        timings: {
          finished_at: '2023-03-04T00:54:46.218046Z'
        }
      },
      jobs: [
        {
          job: {
            name: 'do-thing',
            depends: undefined,
            timings: {
              finished_at: '2023-03-04T00:54:46.218046Z'
            }
          },
          steps: [
            {
              name: 'go-builder',
              depends: undefined,
              timings: {
                finished_at: '2023-03-04T00:54:46.218046Z'
              }
            },
            {
              name: 'python-builder',
              depends: [
                {
                  step_name: 'go-builder'
                }
              ],
              timings: {
                finished_at: '2023-03-04T00:54:46.218046Z'
              }
            }
          ]
        }
      ]
    } as IBuildGraph;

    renderBuildFlow(bgGraph);

    await waitFor(() => {
      expect(screen.getByText('do-thing')).toBeInTheDocument();
    });

    expect(screen.getByText('go-builder')).toBeInTheDocument();
    expect(screen.getByText('python-builder')).toBeInTheDocument();
  });
});
