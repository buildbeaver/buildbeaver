export interface IRunner {
  architecture: string;
  created_at: string;
  deleted_at?: string;
  etag: string;
  id: string;
  labels: string[];
  legal_entity_id: string;
  name: string;
  enabled: boolean;
  operating_system: string;
  software_version: string;
  supported_job_types: string[];
  updated_at: string;
  url: string;
}
