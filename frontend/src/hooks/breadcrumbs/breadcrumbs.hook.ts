import { useLocation } from 'react-router-dom';
import { IBreadcrumb } from './interfaces/breadcrumb.interface';
import { getBreadcrumbsForPath } from './functions/breadcrumbs.functions';

export function useBreadcrumbs(): IBreadcrumb[] {
  const location = useLocation();

  return getBreadcrumbsForPath(location.pathname);
}
