import { IEnvironment } from './environment.interface';

export interface IService {
  name: string;
  image: string;
  docker_authentication?: string;
  environment?: IEnvironment[];
  pull?: string;
}
