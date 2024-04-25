import React from 'react';
import { render, screen } from '@testing-library/react';
import { PollingBanner } from './polling-banner.component';
import { PollingConnectionStatus } from '../../enums/polling-connection-status.enum';
import { PollingBannerContext } from '../../contexts/polling-banner/polling-banner.context';

interface RenderOptions {
  pollingConnectionStatus?: PollingConnectionStatus;
}

const defaultRenderOptions: RenderOptions = {
  pollingConnectionStatus: PollingConnectionStatus.Polling
};

describe('PollingBanner', () => {
  const renderPollingBanner = (renderOptions?: RenderOptions): void => {
    const { pollingConnectionStatus } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <PollingBannerContext.Provider
        value={{ addPollingConnection: () => {}, pollingConnectionStatus: pollingConnectionStatus! }}
      >
        <PollingBanner />
      </PollingBannerContext.Provider>
    );
  };

  it('should not render anything when polling successfully', () => {
    renderPollingBanner();

    expect(screen.queryByText('Network issues detected. Re-establishing connection...')).toBeNull();
    expect(screen.queryByText('Network connection lost')).toBeNull();
  });

  it('should render a warning when polling is retrying', () => {
    renderPollingBanner({ pollingConnectionStatus: PollingConnectionStatus.Retrying });

    expect(screen.getByText('Network issues detected. Re-establishing connection...')).toBeInTheDocument();
  });

  it('should render an error when polling has been abandoned', () => {
    renderPollingBanner({ pollingConnectionStatus: PollingConnectionStatus.Abandoned });

    expect(screen.getByText('Network connection lost')).toBeInTheDocument();
  });
});
