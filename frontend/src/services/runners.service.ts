import { IRunner } from '../interfaces/runner.interface';
import { apiDelete, apiGet, apiPatch, apiPost } from './api.service';
import { IUpdateRunnerRequest } from './requests/update-runner-request.interface';
import { ICreateRunnerRequest } from './requests/create-runner-request.interface';

export async function createRunner(url: string, request: ICreateRunnerRequest): Promise<IRunner> {
  return apiPost<IRunner>(url, request);
}

export async function deleteRunner(runner: IRunner): Promise<void> {
  return apiDelete(runner.url);
}

export async function fetchRunner(url: string): Promise<IRunner> {
  return apiGet<IRunner>(url);
}

export async function updateRunner(runner: IRunner, request: IUpdateRunnerRequest): Promise<IRunner> {
  return apiPatch<IRunner>(runner.url, request);
}
