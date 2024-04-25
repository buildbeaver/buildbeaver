import { getBreadcrumbsForPath } from './breadcrumbs.functions';

describe('breadcrumbs-functions', () => {
  describe('#getBreadcrumbsForPath', () => {
    it('should build breadcrumbs for an org build path', () => {
      const path = '/orgs/buildbeaver/repos/playground/builds/1';
      const breadcrumbs = getBreadcrumbsForPath(path);

      expect(breadcrumbs).toStrictEqual([
        {
          label: 'buildbeaver',
          path: '/orgs/buildbeaver'
        },
        {
          label: 'repos',
          path: '/orgs/buildbeaver/repos'
        },
        {
          label: 'playground',
          path: '/orgs/buildbeaver/repos/playground'
        },
        {
          label: 'builds',
          path: '/orgs/buildbeaver/repos/playground/builds'
        },
        {
          label: '1',
          path: '/orgs/buildbeaver/repos/playground/builds/1'
        }
      ]);
    });
  });
});
