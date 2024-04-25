import { ISetupStatus } from '../../interfaces/setup-status.interface';

interface Options {
  builds_run?: boolean;
  buildbeaver_installed?: boolean;
  repos_enabled?: boolean;
  runners_registered?: boolean;
  runners_seen?: boolean;
}

const defaultOptions: Options = {
  builds_run: true,
  buildbeaver_installed: true,
  repos_enabled: true,
  runners_registered: true,
  runners_seen: true
};

export const mockSetupStatus = (options?: Options): ISetupStatus => {
  const { builds_run, buildbeaver_installed, repos_enabled, runners_registered, runners_seen } = {
    ...defaultOptions,
    ...options
  };

  return {
    created_at: '2023-04-23T02:39:59.669129Z',
    id: 'test-setup-status',
    url: '',
    builds_run,
    buildbeaver_installed: buildbeaver_installed,
    repos_enabled,
    runners_registered,
    runners_seen
  } as ISetupStatus;
};
