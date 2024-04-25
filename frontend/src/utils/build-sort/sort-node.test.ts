import { SortNode } from './sort-node';

describe('sort-node', () => {
  describe('addDependency()', () => {
    it('should add a dependency', () => {
      const sortNode = new SortNode('python-builder');

      expect(sortNode.hasNoDependencies).toBeTruthy();
      expect(sortNode.dependencies).toEqual([]);

      sortNode.addDependency('go-builder');

      expect(sortNode.hasNoDependencies).toBeFalsy();
      expect(sortNode.dependencies).toHaveLength(1);
      expect(sortNode.dependencies).toEqual(['go-builder']);
    });
  });

  describe('addDependent()', () => {
    it('should add a dependent', () => {
      const sortNode = new SortNode('go-builder');

      expect(sortNode.hasNoDependents).toBeTruthy();
      expect(sortNode.dependents).toEqual([]);

      sortNode.addDependent('python-builder');

      expect(sortNode.hasNoDependents).toBeFalsy();
      expect(sortNode.dependents).toHaveLength(1);
      expect(sortNode.dependents).toEqual(['python-builder']);
    });
  });
});
