import { ILegalEntity } from '../interfaces/legal-entity.interface';
import { LegalEntityType } from '../enums/legal-entity-type.enum';

export function isOrg(legalEntity: ILegalEntity): boolean {
  return getTypeForLegalEntity(legalEntity) === LegalEntityType.Orgs;
}

export function getTypeForLegalEntity(legalEntity: ILegalEntity): LegalEntityType {
  return legalEntity.type === 'company' ? LegalEntityType.Orgs : LegalEntityType.Users;
}
