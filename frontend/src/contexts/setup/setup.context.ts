import React from 'react';

interface Context {
  /**
   * True if the user is at the setup wizard path
   */
  isInSetupContext: boolean;

  /**
   * The path to the setup wizard for the current legal entity.
   */
  setupPath: string;

  /**
   * The URL to fetch setup status from for the current legal entity.
   */
  setupUrl: string;
}

export const SetupContext = React.createContext<Context>({} as Context);
