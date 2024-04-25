export interface ILogDescriptor {
  created_at: string;
  data_url: string;
  etag: string;
  id: string;
  parent_log_id?: string;
  resource_id: string;
  sealed: boolean;
  size_bytes: number;
  updated_at: string;
  url: string;
}
