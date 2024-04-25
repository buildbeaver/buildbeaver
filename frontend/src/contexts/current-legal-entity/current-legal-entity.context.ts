import React from 'react';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';

interface Context {
  currentLegalEntity: ILegalEntity;
}

export const CurrentLegalEntityContext = React.createContext<Context>({
  currentLegalEntity: undefined as unknown as ILegalEntity
});
