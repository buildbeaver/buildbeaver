import { useParams } from 'react-router-dom';
import { Config } from '../../config';

export function useRepoUrl(): string {
  const { legal_entity_name, repo_name } = useParams();
  return `${Config.API_BASE}/repos/${legal_entity_name}/${repo_name}`;
}
