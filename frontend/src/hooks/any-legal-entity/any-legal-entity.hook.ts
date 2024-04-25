import { useContext } from 'react';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';
import { IAnyLegalEntity } from '../../interfaces/any-legal-entity.interface';
import { getTypeForLegalEntity } from '../../utils/legal-entity.utils';

export function useAnyLegalEntity(): IAnyLegalEntity {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { selectedLegalEntity } = useContext(SelectedLegalEntityContext);
  const anyLegalEntity = currentLegalEntity ?? selectedLegalEntity;

  return {
    name: anyLegalEntity.name,
    type: getTypeForLegalEntity(anyLegalEntity)
  };
}
