import React from 'react';
import { render, screen } from '@testing-library/react';
import { SetupContext } from '../../contexts/setup/setup.context';
import { RunnerListItem } from './runner-list-item.component';
import { BrowserRouter } from 'react-router-dom';
import { mockRunner } from '../../mocks/models/runner.mock';

interface RenderOptions {
  isInSetupContext: boolean;
}

const defaultRenderOptions: RenderOptions = {
  isInSetupContext: false
};

describe('RunnerListItem', () => {
  const renderRunnerListItem = (renderOptions?: RenderOptions): void => {
    const { isInSetupContext } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <SetupContext.Provider value={{ isInSetupContext, setupPath: '', setupUrl: '' }}>
          <RunnerListItem isFirst={true} isLast={true} runner={mockRunner()} />
        </SetupContext.Provider>
      </BrowserRouter>
    );
  };

  it('should render a runner list item with an edit button', () => {
    renderRunnerListItem();

    expect(screen.getByText('test-org-runner-1')).toBeInTheDocument();
  });
});
