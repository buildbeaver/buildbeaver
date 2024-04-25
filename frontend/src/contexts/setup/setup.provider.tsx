import React, { useContext } from 'react';
import { CurrentLegalEntityContext } from '../current-legal-entity/current-legal-entity.context';
import { makeLegalEntityAbsolutePath } from '../../utils/path.utils';
import { useLocation } from 'react-router-dom';
import { SetupContext } from './setup.context';

/**
 * Provides common props related to legal entity setup to the setup banner and the setup wizard.
 */
export function SetupProvider(props: any): JSX.Element {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const location = useLocation();
  const setupPath = `${makeLegalEntityAbsolutePath(currentLegalEntity)}/setup`;
  const isInSetupContext = location.pathname === setupPath;
  const setupUrl = `${currentLegalEntity.url}/setup-status`;
  const providerValue = {
    isInSetupContext,
    setupPath,
    setupUrl
  };

  return <SetupContext.Provider value={providerValue}>{props.children}</SetupContext.Provider>;
}
