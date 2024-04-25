import React from 'react';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';

interface Context {
  legalEntities: ILegalEntity[];
  getLegalEntityById: (legalEntityId: string) => Promise<ILegalEntity>;
  getLegalEntityByName: (legalEntityName: string) => Promise<ILegalEntity>;
}

export const LegalEntitiesContext = React.createContext<Context>({} as Context);
