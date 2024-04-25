import { ISecret } from '../../interfaces/secret.interface';

export const mockSecret = (): ISecret => {
  return {
    id: 'test_secret',
    name: 'Test secret'
  } as ISecret;
};
