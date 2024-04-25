import React from 'react';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';

interface Context {
  selectedLegalEntity: ILegalEntity;
  selectLegalEntity: (legalEntity: ILegalEntity) => void;
}

export const SelectedLegalEntityContext = React.createContext<Context>({
  selectedLegalEntity: {} as ILegalEntity,
  selectLegalEntity: () => {}
});
