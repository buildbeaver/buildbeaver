import React from 'react';
import { PollingConnection } from '../../models/polling-connection.model';
import { PollingConnectionStatus } from '../../enums/polling-connection-status.enum';

interface Context {
  addPollingConnection: (pollingConnection: PollingConnection) => void;
  pollingConnectionStatus: PollingConnectionStatus;
}

export const PollingBannerContext = React.createContext<Context>({} as Context);
