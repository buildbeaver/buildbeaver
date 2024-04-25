import { IArtifactDefinition } from '../../interfaces/artifact-definition.interface';

export const mockArtifact = (): IArtifactDefinition => {
  return {
    group_name: 'test-group',
    name: 'test-artifact.go',
    path: 'foo/bar/test-artifact.go',
    size: 13077
  } as IArtifactDefinition;
};
