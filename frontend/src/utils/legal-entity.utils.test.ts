import { ILegalEntity } from '../interfaces/legal-entity.interface';
import { getTypeForLegalEntity, isOrg } from './legal-entity.utils';
import { LegalEntityType } from '../enums/legal-entity-type.enum';

describe('legal-entity.utils', () => {
  describe('isOrg', () => {
    it('should return true for a company legal entity', () => {
      const legalEntity = {
        type: 'company'
      } as ILegalEntity;

      expect(isOrg(legalEntity)).toBeTruthy();
    });

    it('should return false for a person legal entity', () => {
      const legalEntity = {
        type: 'person'
      } as ILegalEntity;

      expect(isOrg(legalEntity)).toBeFalsy();
    });
  });

  describe('getTypeForLegalEntity', () => {
    it('should return LegalEntityType.Orgs for a company legal entity', () => {
      const legalEntity = {
        type: 'company'
      } as ILegalEntity;

      expect(getTypeForLegalEntity(legalEntity)).toBe(LegalEntityType.Orgs);
    });

    it('should return LegalEntityType.Users for a person legal entity', () => {
      const legalEntity = {
        type: 'person'
      } as ILegalEntity;

      expect(getTypeForLegalEntity(legalEntity)).toBe(LegalEntityType.Users);
    });
  });
});
