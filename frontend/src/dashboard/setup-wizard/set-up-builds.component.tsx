import React, { useContext } from 'react';
import { NavLink } from 'react-router-dom';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { makeLegalEntityAbsolutePath } from '../../utils/path.utils';

export function SetUpBuilds(): JSX.Element {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const legalEntityAbsolutePath = makeLegalEntityAbsolutePath(currentLegalEntity);
  const reposPath = `${legalEntityAbsolutePath}/repos`;
  const runnersPath = `${legalEntityAbsolutePath}/runners`;

  return (
    <div>
      <p>Before continuing now is a good time to:</p>
      <br />
      <ol className="list-inside list-decimal">
        <li>
          Review the repos have been enabled for using BuildBeaver for{' '}
          <NavLink className="text-blue-400" to={reposPath}>
            here
          </NavLink>
          .
        </li>
        <li>
          Review the runners that have been registered{' '}
          <NavLink className="text-blue-400" to={runnersPath}>
            here
          </NavLink>
          .
        </li>
      </ol>
      <br />
      <p>
        Now that you have enabled your repositories and registered your runners, make a commit against an enabled
        repository to see them on your dashboard.
      </p>
    </div>
  );
}
