import { apiGet } from './api.service';
import { IRootDocument } from '../interfaces/root-document.interface';
import { Config } from '../config';
import { ILegalEntity } from '../interfaces/legal-entity.interface';
import { IResourceResponse } from './responses/resource-response.interface';
import { ISearchResponse } from './responses/search-response';

export async function fetchRootDocument(): Promise<IRootDocument> {
  return apiGet<IRootDocument>(Config.API_BASE);
}

export async function fetchLegalEntities(url: string): Promise<IResourceResponse<ILegalEntity>> {
  return apiGet<IResourceResponse<ILegalEntity>>(url);
}

export async function fetchLegalEntity(url: string): Promise<ILegalEntity> {
  return apiGet<ILegalEntity>(url);
}

export async function search(query: string): Promise<ISearchResponse> {
  return apiGet<ISearchResponse>(`${Config.API_BASE}/search?limit=30&q=${query}`);
}
