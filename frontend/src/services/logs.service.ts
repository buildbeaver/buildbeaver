import { apiGet } from './api.service';
import { ILogDescriptor } from '../interfaces/log-descriptor.interface';
import { IStructuredLog } from '../interfaces/structured-log.interface';

export async function fetchLogDescriptor(url: string): Promise<ILogDescriptor> {
  return apiGet<ILogDescriptor>(url);
}

export async function fetchLogs(url: string, start: number): Promise<IStructuredLog[]> {
  return apiGet(`${url}?start=${start}`);
}
