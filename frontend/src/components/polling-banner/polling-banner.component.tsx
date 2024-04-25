import React, { useContext } from 'react';
import { IoCloseCircleSharp, IoWarning } from 'react-icons/io5';
import { PollingBannerContext } from '../../contexts/polling-banner/polling-banner.context';
import { PollingConnectionStatus } from '../../enums/polling-connection-status.enum';

/**
 * Conditionally shows a banner warning the user of connection errors related to polling.
 */
export function PollingBanner(): JSX.Element {
  const { pollingConnectionStatus } = useContext(PollingBannerContext);

  const emptyBanner = (): JSX.Element => {
    return <></>;
  };

  const abandonedBanner = (): JSX.Element => {
    return (
      <div className="flex justify-center bg-amaranthTransparent px-6 py-1 text-amaranth">
        <div className="flex items-center gap-x-1">
          <div className="text-amaranth">
            <IoCloseCircleSharp size={20} />
          </div>
          <span>Network connection lost</span>
        </div>
      </div>
    );
  };

  const retryingBanner = (): JSX.Element => {
    return (
      <div className="flex justify-center bg-flushOrangeTransparent px-6 py-4 text-flushOrange">
        <div className="flex animate-pulse items-center gap-x-1">
          <div>
            <IoWarning size={20} />
          </div>
          <span>Network issues detected. Re-establishing connection...</span>
        </div>
      </div>
    );
  };

  if (pollingConnectionStatus === PollingConnectionStatus.Polling) {
    return emptyBanner();
  }

  if (pollingConnectionStatus === PollingConnectionStatus.Abandoned) {
    return abandonedBanner();
  }

  return retryingBanner();
}
