import { ILegalEntity } from '../../interfaces/legal-entity.interface';

export const mockOrg = (): ILegalEntity => {
  return {
    name: 'test-org',
    repo_search_url: '',
    type: 'company'
  } as ILegalEntity;
};
