import { useParams } from 'react-router-dom';
import { Config } from '../../config';

export function useSecretUrl(): string {
  const { secret_id } = useParams();

  return `${Config.API_BASE}/secrets/secret:${secret_id}`;
}
