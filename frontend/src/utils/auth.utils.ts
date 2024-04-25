import { getCookie, removeCookie } from 'typescript-cookie';
import { localStorageUtils } from './storage/local-storage.utils';
import { sessionStorageUtils } from './storage/session-storage.utils';

/**
 * Cookie will not be present if either:
 *  a) The user never authenticated, or
 *  b) The user authenticated but their cookie has since expired
 */
export function isAuthenticated(): boolean {
  return getCookie('buildbeaver') !== undefined;
}

export function clearSensitiveData(): void {
  removeCookie('buildbeaver');
  localStorageUtils.clear();
  sessionStorageUtils.clear();
}
