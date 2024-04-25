import { IRunner } from '../../interfaces/runner.interface';

export const mockRunner = (): IRunner => {
  return {
    architecture: 'amd64',
    created_at: '2023-04-23T02:40:02.496375Z',
    id: 'test-runner',
    name: 'test-org-runner-1',
    operating_system: 'windows',
    software_version: '0.1.0'
  } as IRunner;
};
