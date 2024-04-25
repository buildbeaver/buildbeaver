import React from 'react';
import { NavLink } from 'react-router-dom';
import { IoCalendarClearOutline } from 'react-icons/io5';
import { ISecret } from '../../interfaces/secret.interface';
import { createdAt } from '../../utils/build.utils';

interface Props {
  secret: ISecret;
}

/**
 * Constructs an individual Repo Secret list item
 */
export function RepoSecretListItem(props: Props): JSX.Element {
  const { secret } = props;
  const reactRouterSafeId = secret.id.replace('secret:', '');

  return (
    <NavLink className="flex cursor-pointer justify-between text-sm text-gray-600 hover:bg-gray-100" to={reactRouterSafeId}>
      <div className="flex min-w-0 grow">
        <div className={`w-[4px] min-w-[4px]`}></div>
        <div className="flex w-[60%] min-w-[60%] gap-x-2 p-3">
          <div className="flex min-w-0 flex-col gap-y-1">
            <div className="... truncate" title={secret.name}>
              {secret.name}
            </div>
          </div>
        </div>
      </div>
      <div className="flex w-[150px] min-w-[150px] flex-col justify-center gap-y-1 p-3 text-xs">
        <div className="flex items-center gap-x-2" title="Updated">
          <IoCalendarClearOutline className="ml-[1px]" size={14} />
          {createdAt(secret.updated_at)}
        </div>
      </div>
    </NavLink>
  );
}
