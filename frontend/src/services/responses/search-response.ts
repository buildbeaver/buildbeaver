import { IResourceResponse } from './resource-response.interface';

export interface ISearchResponse extends IResourceResponse<any> {
  results: IResourceResponse<any>[];
}
