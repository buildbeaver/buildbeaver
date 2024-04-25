import { apiGet } from './api.service';
import { IJobGraph } from '../interfaces/job-graph.interface';

export async function fetchJobGraph(url: string): Promise<IJobGraph> {
  return apiGet<IJobGraph>(url);
}
