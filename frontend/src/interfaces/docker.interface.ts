import { IAWSAuth } from './aws-auth.interface';

export interface IDocker {
  aws_auth?: IAWSAuth;
  image: string;
  pull: string;
  shell?: string;
}
