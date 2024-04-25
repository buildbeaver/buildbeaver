import React, { useContext, useEffect, useState } from 'react';
import { RootContext } from '../root/root.context';
import { LegalEntitiesContext } from './legal-entities.context';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';
import { fetchLegalEntities, fetchLegalEntity } from '../../services/root.service';
import { Loading } from '../../components/loading/loading.component';
import { Config } from '../../config';
import { Navigate, useLocation } from 'react-router-dom';
import { ToasterContext } from '../toaster/toaster.context';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { getStructuredErrorMessage } from '../../utils/error.utils';

export function LegalEntitiesProvider(props: any): JSX.Element {
  const [legalEntities, setLegalEntities] = useState<ILegalEntity[]>([]);
  const [failed, setFailed] = useState(false);
  const [loading, setLoading] = useState(true);
  const rootDocument = useContext(RootContext);
  const { pathname } = useLocation();
  const { toastError } = useContext(ToasterContext);
  const legalEntityType = pathname.split('/')[1];

  const getLegalEntityById = async (legalEntityId: string): Promise<ILegalEntity> => {
    const legalEntity = legalEntities.find((cachedLegalEntity) => cachedLegalEntity.id === legalEntityId);

    if (legalEntity) {
      return legalEntity;
    }

    return await fetchLegalEntity(`${Config.API_BASE}/legal-entities/${legalEntityId}`).then((legalEntity) => {
      setLegalEntities([...legalEntities, legalEntity]);
      return legalEntity;
    });
  };

  const getLegalEntityByName = async (legalEntityName: string): Promise<ILegalEntity> => {
    const legalEntity = legalEntities.find((cachedLegalEntity) => cachedLegalEntity.name === legalEntityName);

    if (legalEntity) {
      return legalEntity;
    }

    return await fetchLegalEntity(`${Config.API_BASE}/${legalEntityType}/${legalEntityName}`).then((legalEntity) => {
      setLegalEntities([...legalEntities, legalEntity]);
      return legalEntity;
    });
  };

  useEffect(() => {
    const getLegalEntities = async () => {
      await fetchLegalEntities(rootDocument.legal_entities_url)
        .then((response) => {
          if (response.results) {
            setLegalEntities(response.results);
          } else {
            toastError('No users or companies were found');
            setFailed(true);
          }
        })
        .catch((error: IStructuredError) => {
          toastError(getStructuredErrorMessage(error, 'Failed to fetch user or company information'));
          setFailed(true);
        })
        .finally(() => {
          setLoading(false);
        });
    };

    getLegalEntities();
  }, [rootDocument]);

  if (failed) {
    return <Navigate to={'/sign-out'} />;
  }

  if (loading) {
    return <Loading />;
  }

  const providerValue = {
    legalEntities: legalEntities,
    getLegalEntityById,
    getLegalEntityByName
  };

  return <LegalEntitiesContext.Provider value={providerValue}>{props.children}</LegalEntitiesContext.Provider>;
}
