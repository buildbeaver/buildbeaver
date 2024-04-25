import { IRepo } from '../interfaces/repo.interface';
import { apiGet, apiPatch } from './api.service';
import { IUpdateRepoRequest } from './requests/update-repo-request.interface';

export async function fetchRepo(url: string): Promise<IRepo> {
  return apiGet<IRepo>(url);
}

export async function updateRepo(repo: IRepo, request: IUpdateRepoRequest): Promise<IRepo> {
  return apiPatch<IRepo>(repo.url, request);
}
