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
        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nam finibus nunc in leo pulvinar tristique. Donec purus ligula,
        egestas accumsan est sed, maximus vehicula odio. Aliquam placerat sollicitudin augue. Donec vel vulputate mi. Nulla luctus
        neque sed mi sollicitudin aliquam. Donec vel justo quis dui dapibus hendrerit nec eget ipsum. Pellentesque mauris purus,
        ullamcorper id tristique non, mollis quis odio. Sed id molestie felis, eget rutrum ante. Duis pulvinar lacus purus. Fusce
        gravida egestas tellus sit amet fermentum. Aenean nec placerat nulla.
      </p>
      <br />
      <p>
        Aenean lorem ante, consectetur dapibus lacus nec, varius porttitor erat. Nunc at orci leo. Duis malesuada sapien pharetra
        malesuada facilisis. Cras vitae nulla nibh. Nam nec eleifend lacus. Maecenas nec ex sollicitudin, vehicula justo nec,
        ornare felis. Curabitur sit amet sapien vel purus fermentum finibus. Sed id faucibus nunc.
      </p>
    </div>
  );
}
