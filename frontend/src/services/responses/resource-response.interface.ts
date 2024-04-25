import { ResourceKind } from '../../enums/resource-kind.enum';

/**
 * IResourceResponse describes the response from our resource APIs
 */
export interface IResourceResponse<Type> {
  kind?: ResourceKind;
  next_url: string;
  prev_url: string;
  results: Type[];
}
