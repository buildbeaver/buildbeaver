export interface IStorage {
  clear(): void;
  getItem<T>(key: string): T | null;
  setItem<T>(key: string, value: T): void;
}
