import * as resourceListHook from '../../hooks/resources/resource-list.hook';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IUseStaticResources } from '../../hooks/resources/resource-list.hook';

/**
 * Provides a mock implementation of our generic useStaticResourceList for use in any UI testing.
 */
export function mockUseStaticResourceList<Type>(kind: ResourceKind, options?: Partial<IUseStaticResources<Type>>): void {
  const defaultOptions: IUseStaticResources<Type> = {
    loading: false,
    error: undefined,
    response: {
      kind: kind,
      next_url: '',
      prev_url: '',
      results: new Array<Type>()
    },
    refresh: () => {}
  };

  const { loading, error, response, refresh } = {
    ...defaultOptions,
    ...options
  };

  // TODO: Investigate if jest can spyOn generics where we can specify the type here.
  // https://github.com/facebook/jest/pull/12489 might provide a result here
  jest.spyOn(resourceListHook, 'useStaticResourceList').mockImplementation(() => {
    return {
      loading,
      error,
      response,
      refresh
    };
  });
}
