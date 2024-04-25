import { ISecret } from '../interfaces/secret.interface';
import { apiDelete, apiPost, apiPatch, apiGet } from './api.service';

export async function createSecret(url: string, name: string, value: string): Promise<ISecret> {
  return apiPost<ISecret>(url, { name, value });
}

export async function deleteSecret(secret: ISecret): Promise<void> {
  await apiDelete(secret.url);
}

export async function fetchSecret(url: string): Promise<ISecret> {
  return apiGet<ISecret>(url);
}

export async function updateSecret(secret: ISecret, name: string, value: string): Promise<void> {
  await apiPatch(secret.url, { name, value });
}
