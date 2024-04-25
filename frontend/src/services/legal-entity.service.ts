import { apiGet } from './api.service';
import { IBuildsSummary } from '../interfaces/builds-summary.interface';
import { ISetupStatus } from '../interfaces/setup-status.interface';

export async function fetchBuildSummary(url: string): Promise<IBuildsSummary> {
  return apiGet<IBuildsSummary>(url);
}

export async function fetchSetupStatus(url: string): Promise<ISetupStatus> {
  return apiGet<ISetupStatus>(url);
}
