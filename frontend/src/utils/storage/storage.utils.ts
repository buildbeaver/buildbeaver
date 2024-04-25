import { IStorage } from './storage.interface';

/**
 * Wraps existing browser storage to support storage of additional types as JSON strings.
 */
export function storageUtils(storage: Storage): IStorage {
  function clear(): void {
    return storage.clear();
  }

  function getItem<T>(key: string): T | null {
    const item = storage.getItem(key);

    if (item !== null) {
      try {
        return JSON.parse(item);
      } catch (error: unknown) {
        console.error(`Failed to parse stored value with key "${key}"`, error);
      }
    }

    return null;
  }

  function setItem<T>(key: string, value: T): void {
    storage.setItem(key, JSON.stringify(value));
  }

  return {
    clear,
    getItem,
    setItem
  };
}
