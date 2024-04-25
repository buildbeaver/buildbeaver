import * as anyLegalEntityHook from '../../hooks/any-legal-entity/any-legal-entity.hook';
import { LegalEntityType } from '../../enums/legal-entity-type.enum';
import { IAnyLegalEntity } from '../../interfaces/any-legal-entity.interface';

const defaultAnyLegalEntity: IAnyLegalEntity = {
  name: 'test-org',
  type: LegalEntityType.Orgs
};

export function mockUseAnyLegalEntity(anyLegalEntity?: IAnyLegalEntity): void {
  const { name, type } = {
    ...defaultAnyLegalEntity,
    ...anyLegalEntity
  };

  jest.spyOn(anyLegalEntityHook, 'useAnyLegalEntity').mockImplementation(() => {
    return {
      name,
      type
    };
  });
}
