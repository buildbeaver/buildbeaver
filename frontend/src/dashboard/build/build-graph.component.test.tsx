import React from 'react';
import { render, screen } from '@testing-library/react';
import { mockBuildGraph } from '../../mocks/models/build-graph.mock';
import { BrowserRouter } from 'react-router-dom';
import { mockUseSetupStatus } from '../../mocks/hooks/useSetupStatus.mock';
import { mockUseIndirectedJob } from '../../mocks/hooks/useIndirectedJob.mock';
import { mockJobGraph } from '../../mocks/models/jobGraph.mock';
import { BuildGraph } from './build-graph.component';
import { mockUseBuild } from '../../mocks/hooks/useBuild.mock';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { mockOrg } from '../../mocks/models/org.mock';

jest.mock('react-router-dom', () => ({
  ...jest.requireActual('react-router-dom'),
  useLocation: () => ({
    pathname: '/orgs/test-org/repos/test-repo/builds/1/test-job/log'
  })
}));
jest.mock('../../components/build-flow/build-flow.component', () => ({ BuildFlow: () => 'Mocked DAG' }));

interface RenderOptions {
  buildGraph?: IBuildGraph;
}

const defaultRenderOptions: RenderOptions = {
  buildGraph: mockBuildGraph()
};

describe('BuildGraph', () => {
  const renderBuildGraph = (renderOptions?: RenderOptions): void => {
    const { buildGraph } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    mockUseIndirectedJob();
    mockUseSetupStatus();
    mockUseBuild(buildGraph!);

    render(
      <BrowserRouter>
        <CurrentLegalEntityContext.Provider value={{ currentLegalEntity: mockOrg() }}>
          <BuildGraph />
        </CurrentLegalEntityContext.Provider>
      </BrowserRouter>
    );
  };

  it('should render a DAG for a build', () => {
    renderBuildGraph({ buildGraph: mockBuildGraph({ jobs: [mockJobGraph()] }) });

    expect(screen.getByText('Mocked DAG')).toBeInTheDocument();
  });

  describe('when the job graph array is undefined', () => {
    it('should not render the DAG', () => {
      renderBuildGraph({ buildGraph: mockBuildGraph({ jobs: undefined }) });

      expect(screen.queryByText('Mocked DAG')).toBeNull();
    });
  });

  describe('when the job graph array is empty', () => {
    it('should not render the DAG', () => {
      renderBuildGraph({ buildGraph: mockBuildGraph({ jobs: [] }) });

      expect(screen.queryByText('Mocked DAG')).toBeNull();
    });
  });
});
