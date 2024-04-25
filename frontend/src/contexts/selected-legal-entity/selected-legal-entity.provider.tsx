import React, { useContext, useEffect, useState } from 'react';
import { LegalEntitiesContext } from '../legal-entities/legal-entities.context';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';
import { StorageKeys } from '../../utils/storage/storage.keys';
import { localStorageUtils } from '../../utils/storage/local-storage.utils';
import { SelectedLegalEntityContext } from './selected-legal-entity.context';
import { Loading } from '../../components/loading/loading.component';

export function SelectedLegalEntityProvider(props: any): JSX.Element {
  const { legalEntities } = useContext(LegalEntitiesContext);
  const [selectedLegalEntity, setSelectedLegalEntity] = useState<ILegalEntity | undefined>();

  useEffect(() => {
    if (legalEntities.length > 0) {
      const storedName = localStorageUtils.getItem<string>(StorageKeys.CURRENT_LEGAL_ENTITY);

      if (storedName !== null) {
        const matchingEntity = legalEntities.find((legalEntity) => legalEntity.name === storedName);

        if (matchingEntity) {
          // Preserves the current legal entity between refreshes
          selectLegalEntity(matchingEntity);
          return;
        }
      }

      // Fall back to the first entity we found
      selectLegalEntity(legalEntities[0]);
    }
  }, [legalEntities]);

  const selectLegalEntity = (legalEntity: ILegalEntity) => {
    setSelectedLegalEntity(legalEntity);
    localStorageUtils.setItem<string>(StorageKeys.CURRENT_LEGAL_ENTITY, legalEntity.name);
  };

  if (selectedLegalEntity) {
    const providerValue = {
      selectedLegalEntity,
      selectLegalEntity
    };

    return <SelectedLegalEntityContext.Provider value={providerValue}>{props.children}</SelectedLegalEntityContext.Provider>;
  }

  return <Loading />;
}
