import React, { useContext } from 'react';
import { IoWarning } from 'react-icons/io5';
import { NavLink } from 'react-router-dom';
import { useSetupStatus } from '../../hooks/setup-status/setup-status.hook';
import { SetupContext } from '../../contexts/setup/setup.context';

export function SetupBanner(): JSX.Element {
  const { isInSetupContext, setupPath, setupUrl } = useContext(SetupContext);
  const { setupStatus, setupStatusLoading } = useSetupStatus(setupUrl);
  const isSetupRequired =
    !!setupStatus &&
    [setupStatus.builds_run, setupStatus.buildbeaver_installed, setupStatus.repos_enabled, setupStatus.runners_registered].some(
      (step) => !step
    );

  if (setupStatusLoading || !isSetupRequired) {
    return <></>;
  }

  return (
    <div className="flex items-center justify-center gap-x-1 bg-flushOrangeTransparent px-6 py-4 text-flushOrange">
      <div>
        <IoWarning size={20} />
      </div>
      {isInSetupContext ? (
        <div>Additional setup is required</div>
      ) : (
        <div>
          Additional setup is required. Click{' '}
          <NavLink className="cursor-pointer text-curiousBlue" to={setupPath}>
            here
          </NavLink>{' '}
          to set up BuildBeaver.
        </div>
      )}
    </div>
  );
}
