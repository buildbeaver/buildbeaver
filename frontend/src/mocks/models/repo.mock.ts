import { IRepo } from '../../interfaces/repo.interface';

export const mockRepo = (): IRepo => {
  return {
    id: 'test-repo',
    default_branch: 'main',
    description: 'A repo for testing',
    legal_entity_id: 'legal-entity:1527916f-6d25-49b8-b159-14f053eb7c7g',
    name: 'Test repo',
    secrets_url: ''
  } as IRepo;
};
