import React, { useContext, useEffect, useState } from 'react';
import { CurrentLegalEntityContext } from './current-legal-entity.context';
import { Loading } from '../../components/loading/loading.component';
import { SelectedLegalEntityContext } from '../selected-legal-entity/selected-legal-entity.context';
import { useParams } from 'react-router-dom';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';
import { LegalEntitiesContext } from '../legal-entities/legal-entities.context';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { StructuredError } from '../../components/structured-error/structured-error.component';

export function CurrentLegalEntityProvider(props: any): JSX.Element {
  const { getLegalEntityByName } = useContext(LegalEntitiesContext);
  const { selectedLegalEntity } = useContext(SelectedLegalEntityContext);
  const { legal_entity_name } = useParams();
  const [currentLegalEntity, setCurrentLegalEntity] = useState<ILegalEntity | undefined>();
  const [currentLegalEntityError, setCurrentLegalEntityError] = useState<IStructuredError | undefined>();

  // TODO: What if we are at /users instead of /orgs and the current legal entity is an org (for example)

  useEffect(() => {
    const runGetLegalEntity = async (): Promise<void> => {
      if (legal_entity_name) {
        await getLegalEntityByName(legal_entity_name)
          .then((response) => {
            setCurrentLegalEntity(response);
          })
          .catch((error: IStructuredError) => {
            setCurrentLegalEntityError(error);
          });
      } else {
        // No legal_entity_name in the current route, fall back to the selected legal entity
        setCurrentLegalEntity(selectedLegalEntity);
      }
    };

    runGetLegalEntity();
  }, [selectedLegalEntity]);

  if (currentLegalEntityError) {
    return <StructuredError error={currentLegalEntityError} handleNotFound={true} />;
  }

  if (currentLegalEntity) {
    const providerValue = {
      currentLegalEntity
    };

    return <CurrentLegalEntityContext.Provider value={providerValue}>{props.children}</CurrentLegalEntityContext.Provider>;
  }

  return <Loading />;
}
