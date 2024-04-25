export interface IStructuredError {
  message?: string;
  serverError?: {
    code: string;
    details: unknown;
    http_status_code: number;
    message: string;
  };
  statusCode: number;
  statusText: string;
}
