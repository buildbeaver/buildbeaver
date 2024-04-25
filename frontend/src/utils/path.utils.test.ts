import { removeLastPathPart, replacePathParts } from './path.utils';

describe('path', () => {
  describe('removeLastPathPart', () => {
    it('should remove the last path part', () => {
      expect(removeLastPathPart('/secrets/1')).toBe('/secrets');
    });
  });

  describe('replacePathParts', () => {
    it('should replace multiple path parts', () => {
      const path = '/builds/1/js-test/log';
      const partReplacements = [
        { positionFromEnd: 1, replacement: 'artifacts' },
        { positionFromEnd: 2, replacement: 'package' }
      ];
      const replacedPath = replacePathParts(path, partReplacements);

      expect(replacedPath).toBe('/builds/1/package/artifacts');
    });

    it('should replace when the path is not prefixed with a separator', () => {
      const path = 'builds/1/js-test/log';
      const partReplacements = [{ positionFromEnd: 1, replacement: 'artifacts' }];
      const replacedPath = replacePathParts(path, partReplacements);

      expect(replacedPath).toBe('builds/1/js-test/artifacts');
    });

    it('should throw if trying to replace an out of bounds index', () => {
      const path = '/builds/1/js-test/log';
      const partReplacements = [{ positionFromEnd: 5, replacement: 'repos' }];

      expect(() => replacePathParts(path, partReplacements)).toThrow(
        'Cannot replace part in path at position from end 5, number of parts is 4'
      );
    });
  });
});
