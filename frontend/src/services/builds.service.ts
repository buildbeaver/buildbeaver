import { apiGet, apiPost } from './api.service';
import { IBuildGraph } from '../interfaces/build-graph.interface';
import { ICreateBuildRequest } from './requests/create-build-request.interface';

export async function createBuild(url: string, request: ICreateBuildRequest): Promise<IBuildGraph> {
  return apiPost<IBuildGraph>(url, request);
}

export async function fetchBuild(url: string): Promise<IBuildGraph> {
  return apiGet<IBuildGraph>(url);
}
