export interface IAWSAuth {
  aws_region?: unknown;
  aws_access_key_id: {
    value: string;
    value_from_secret: string;
  };
  aws_secret_access_key: {
    value: string;
    value_from_secret: string;
  };
}
