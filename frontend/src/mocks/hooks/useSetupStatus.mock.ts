import * as setupStatusHook from '../../hooks/setup-status/setup-status.hook';
import { mockSetupStatus } from '../models/setup-status.mock';
import { IUseSetupStatus } from '../../hooks/setup-status/setup-status.hook';

export function mockUseSetupStatus(useSetupStatus?: IUseSetupStatus): void {
  const mockUseSetupStatus = {
    setupStatus: mockSetupStatus(),
    setupStatusError: undefined,
    setupStatusLoading: false,
    setupStatusRefreshing: false,
    refreshSetupStatus: jest.fn(),
    ...useSetupStatus
  };

  jest.spyOn(setupStatusHook, 'useSetupStatus').mockImplementation(() => {
    return mockUseSetupStatus;
  });
}
