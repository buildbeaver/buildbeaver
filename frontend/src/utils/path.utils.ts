import { ILegalEntity } from '../interfaces/legal-entity.interface';
import { getTypeForLegalEntity } from './legal-entity.utils';

interface IPartReplacement {
  positionFromEnd: number;
  replacement: string;
}

/**
 * Provides the absolute base path to routes for an entity based on its type.
 * @example '/orgs/buildbeaver'
 */
export function makeLegalEntityAbsolutePath(legalEntity: ILegalEntity): string {
  return `/${getTypeForLegalEntity(legalEntity)}/${legalEntity.name}`;
}

export function removeLastPathPart(path: string): string {
  return path.substring(0, path.lastIndexOf('/'));
}

export function replacePathParts(path: string, partReplacements: IPartReplacement[]): string {
  const pathParts = path.split('/');
  const numberOfParts = pathParts.length;
  const lastPositionToReplace = pathParts[0] === '' ? numberOfParts - 1 : numberOfParts;

  for (const partReplacement of partReplacements) {
    const { positionFromEnd, replacement } = partReplacement;

    if (positionFromEnd > lastPositionToReplace) {
      throw new Error(
        `Cannot replace part in path at position from end ${positionFromEnd}, number of parts is ${lastPositionToReplace}`
      );
    }

    pathParts[numberOfParts - positionFromEnd] = replacement;
  }

  return pathParts.join('/');
}
