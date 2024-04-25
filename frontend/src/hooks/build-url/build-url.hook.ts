import { useParams } from 'react-router-dom';
import { Config } from '../../config';

export function useBuildUrl(): string {
  const { build_name, legal_entity_name, repo_name } = useParams();

  return `${Config.API_BASE}/repos/${legal_entity_name}/${repo_name}/builds/${build_name}`;
}
