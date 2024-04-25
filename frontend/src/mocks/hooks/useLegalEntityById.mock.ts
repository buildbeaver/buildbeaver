import * as legalEntityByIdHook from '../../hooks/legal-entity-by-id/legal-entity-by-id.hook';
import { ILegalEntity } from '../../interfaces/legal-entity.interface';

const defaultLegalEntity: ILegalEntity = {
  name: 'test-org',
  type: 'company'
} as ILegalEntity;

export function mockUseLegalEntityById(legalEntity?: ILegalEntity): void {
  const { name, type } = {
    ...defaultLegalEntity,
    ...legalEntity
  };

  jest.spyOn(legalEntityByIdHook, 'useLegalEntityById').mockImplementation(() => {
    return {
      legalEntity: {
        name,
        type
      } as ILegalEntity,
      legalEntityError: undefined
    };
  });
}
