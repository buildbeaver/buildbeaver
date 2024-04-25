import React, { useEffect, useState } from 'react';
import { PollingBannerContext } from './polling-banner.context';
import { PollingConnectionStatus } from '../../enums/polling-connection-status.enum';
import { useTick } from '../../hooks/tick/tick.hook';
import { PollingConnection } from '../../models/polling-connection.model';

/**
 * Determines global polling connection status based on any number of provided polling connections.
 */
export function PollingBannerProvider(props: any): JSX.Element {
  const [hasRetryingConnections, setHasRetryingConnections] = useState(false);
  const tick = useTick(2000, hasRetryingConnections);
  const [connections, setConnections] = useState<PollingConnection[]>([]);
  const [status, setStatus] = useState(PollingConnectionStatus.Polling);

  const addPollingConnection = (pollingConnection: PollingConnection): void => {
    setConnections([...connections, pollingConnection]);
    setHasRetryingConnections(true);
  };

  useEffect(() => {
    // Clears out stale healthy connections every time this effect runs
    const activeConnections = connections.filter((connection) => connection.status !== PollingConnectionStatus.Polling);

    let pollingConnectionStatus: PollingConnectionStatus;

    if (activeConnections.length === 0) {
      pollingConnectionStatus = PollingConnectionStatus.Polling;
      setHasRetryingConnections(false);
    } else if (activeConnections.some((connection) => connection.status === PollingConnectionStatus.Retrying)) {
      // If any connections are retrying we want to show we are retrying, even if other connections are abandoned
      pollingConnectionStatus = PollingConnectionStatus.Retrying;
      setHasRetryingConnections(true);
    } else {
      pollingConnectionStatus = PollingConnectionStatus.Abandoned;
      setHasRetryingConnections(false);
    }

    setStatus(pollingConnectionStatus);
    setConnections(activeConnections);
  }, [tick]);

  return (
    <PollingBannerContext.Provider value={{ addPollingConnection, pollingConnectionStatus: status }}>
      {props.children}
    </PollingBannerContext.Provider>
  );
}
