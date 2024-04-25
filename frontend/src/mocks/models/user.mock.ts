import { ILegalEntity } from '../../interfaces/legal-entity.interface';

export const mockUser = (): ILegalEntity => {
  return {
    name: 'test-person',
    repo_search_url: '',
    type: 'person'
  } as ILegalEntity;
};
