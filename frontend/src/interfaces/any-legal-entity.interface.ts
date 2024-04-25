import { LegalEntityType } from '../enums/legal-entity-type.enum';

export interface IAnyLegalEntity {
  name: string;
  type: LegalEntityType;
}
