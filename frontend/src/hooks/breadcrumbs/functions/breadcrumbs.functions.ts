import { IBreadcrumb } from '../interfaces/breadcrumb.interface';

export function getBreadcrumbsForPath(path: string): IBreadcrumb[] {
  const pathParts = path.split('/');

  let breadcrumbPath = pathParts.slice(0, 2).join('/');

  return pathParts.slice(2).map((part) => {
    breadcrumbPath += `/${part}`;

    return {
      label: part,
      path: breadcrumbPath
    };
  });
}
