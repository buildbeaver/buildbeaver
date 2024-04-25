import { useContext, useEffect, useState } from 'react';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';
import { LegalEntitiesContext } from '../../contexts/legal-entities/legal-entities.context';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface IUseLegalEntityById {
  legalEntity?: ILegalEntity;
  legalEntityError?: IStructuredError;
}

/**
 * Wrapper around getting a single legal entity by id. Fetches the legal entity and caches it if we haven't seen it
 * before. Otherwise, just returns it.
 * @param id - The id of the legal entity to fetch.
 */
export function useLegalEntityById(id: string): IUseLegalEntityById {
  const { getLegalEntityById } = useContext(LegalEntitiesContext);
  const [legalEntity, setLegalEntity] = useState<ILegalEntity | undefined>();
  const [legalEntityError, setLegalEntityError] = useState<IStructuredError | undefined>();

  useEffect(() => {
    const runGetLegalEntityById = async (): Promise<void> => {
      await getLegalEntityById(id)
        .then((response) => {
          setLegalEntity(response);
        })
        .catch((error: IStructuredError) => {
          setLegalEntityError(error);
        });
    };

    runGetLegalEntityById();
  }, []);

  return {
    legalEntity,
    legalEntityError
  };
}
