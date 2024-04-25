import { useParams } from 'react-router-dom';
import { Config } from '../../config';
import { useContext } from 'react';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { getTypeForLegalEntity } from '../../utils/legal-entity.utils';

export function useRunnerUrl(): string {
  const { runner_name } = useParams();
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);

  return `${Config.API_BASE}/${getTypeForLegalEntity(currentLegalEntity)}/${currentLegalEntity.name}/runners/${runner_name}`;
}
