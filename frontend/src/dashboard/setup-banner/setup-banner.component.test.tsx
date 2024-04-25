import React from 'react';
import { render, screen } from '@testing-library/react';
import { SetupContext } from '../../contexts/setup/setup.context';
import { BrowserRouter } from 'react-router-dom';
import { SetupBanner } from './setup-banner.component';
import { mockUseSetupStatus } from '../../mocks/hooks/useSetupStatus.mock';
import { IUseSetupStatus } from '../../hooks/setup-status/setup-status.hook';

interface RenderOptions {
  isInSetupContext: boolean;
}

const defaultRenderOptions: RenderOptions = {
  isInSetupContext: false
};

describe('SetupBanner', () => {
  const renderSetupBanner = (renderOptions?: RenderOptions): void => {
    const { isInSetupContext } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <SetupContext.Provider value={{ isInSetupContext, setupPath: '/orgs/test-org/setup', setupUrl: '' }}>
          <SetupBanner />
        </SetupContext.Provider>
      </BrowserRouter>
    );
  };

  it('should render when setup is required', () => {
    mockUseSetupStatus({ setupStatus: { builds_run: false } } as IUseSetupStatus);
    renderSetupBanner();

    expect(screen.getByRole('link', { name: 'here' })).toBeInTheDocument();
  });

  it('should not render anything while on the setup page', () => {
    mockUseSetupStatus();
    renderSetupBanner({ isInSetupContext: true });

    expect(screen.queryByRole('link', { name: 'here' })).toBeNull();
  });

  it('should not render anything while loading', () => {
    mockUseSetupStatus({ setupStatusLoading: true } as IUseSetupStatus);
    renderSetupBanner();

    expect(screen.queryByRole('link', { name: 'here' })).toBeNull();
  });

  it('should not render when setup is complete', () => {
    mockUseSetupStatus();
    renderSetupBanner();

    expect(screen.queryByRole('link', { name: 'here' })).toBeNull();
  });
});
