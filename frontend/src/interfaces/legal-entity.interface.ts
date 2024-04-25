import { IExternalId } from './external-id.interface';

export interface ILegalEntity {
  build_summary_url: string;
  created_at: string;
  email_address: string;
  etag: string;
  external_id: IExternalId;
  external_metadata: string;
  id: string;
  legal_name: string;
  name: string;
  repo_search_url: string;
  runner_search_url: string;
  type: string;
  updated_at: string;
  url: string;
}
