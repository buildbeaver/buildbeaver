import * as logDescriptorHook from '../../hooks/log-descriptor/log-descriptor.hook';
import { IUseLogDescriptor } from '../../hooks/log-descriptor/log-descriptor.hook';
import { ILogDescriptor } from '../../interfaces/log-descriptor.interface';

export function mockUseLogDescriptor(useLogDescriptor?: IUseLogDescriptor): void {
  const mockUseLogDescriptor = {
    logDescriptor: {} as ILogDescriptor,
    logDescriptorError: undefined,
    logDescriptorLoading: false,
    ...useLogDescriptor
  };

  jest.spyOn(logDescriptorHook, 'useLogDescriptor').mockImplementation(() => {
    return mockUseLogDescriptor;
  });
}
