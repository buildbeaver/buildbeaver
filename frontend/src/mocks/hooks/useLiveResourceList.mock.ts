import * as resourceListHook from '../../hooks/resources/resource-list.hook';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IUseLiveResources } from '../../hooks/resources/resource-list.hook';

/**
 * Provides a mock implementation of our generic useLiveResourceList for use in any UI testing.
 */
export function mockUseLiveResourceList<Type>(kind: ResourceKind, options?: Partial<IUseLiveResources<Type>>): void {
  const defaultOptions: IUseLiveResources<Type> = {
    loading: false,
    error: undefined,
    response: {
      kind: kind,
      next_url: '',
      prev_url: '',
      results: new Array<Type>()
    }
  };

  const { loading, error, response } = {
    ...defaultOptions,
    ...options
  };

  // TODO: Investigate if jest can spyOn generics where we can specify the type here.
  // https://github.com/facebook/jest/pull/12489 might provide a result here
  jest.spyOn(resourceListHook, 'useLiveResourceList').mockImplementation(() => {
    return {
      loading,
      error,
      response
    };
  });
}
