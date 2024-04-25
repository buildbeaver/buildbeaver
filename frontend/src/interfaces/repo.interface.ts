import { IExternalId } from './external-id.interface';

export interface IRepo {
  builds_url: string;
  build_search_url: string;
  created_at: string;
  default_branch: string;
  description: string;
  enabled: boolean;
  etag: string;
  external_id: IExternalId;
  external_metadata: string;
  http_url: string;
  id: string;
  legal_entity_id: string;
  link: string;
  name: string;
  private: boolean;
  secrets_url: string;
  ssh_key_secret_id: string;
  ssh_url: string;
  updated_at: string;
  url: string;
}
