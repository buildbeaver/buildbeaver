import React, { useContext } from 'react';
import { SetupBanner } from '../../dashboard/setup-banner/setup-banner.component';
import { PollingBanner } from '../polling-banner/polling-banner.component';
import { NavLink, useParams } from 'react-router-dom';
import { useAnyLegalEntity } from '../../hooks/any-legal-entity/any-legal-entity.hook';
import { INavigationTab } from '../../interfaces/navigation-tab.interface';
import { IoBusinessOutline, IoPersonCircleOutline } from 'react-icons/io5';
import { LegalEntityType } from '../../enums/legal-entity-type.enum';

interface Props {
  navigationTabs: INavigationTab[];
}

export function ContextualHeader(props: Props): JSX.Element {
  const { navigationTabs } = props;
  const { name, type } = useAnyLegalEntity();
  const { build_name, repo_name } = useParams();

  return (
    <div className="flex flex-col">
      <PollingBanner />
      <SetupBanner />
      <div className="flex flex-col border-b bg-gray-100 px-4 pt-2">
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-x-2">
            <div>{type === LegalEntityType.Orgs ? <IoBusinessOutline size={18} /> : <IoPersonCircleOutline size={20} />}</div>
            <h1 className="py-3 text-xl text-gray-600">
              {name && (
                <>
                  <NavLink className="text-blue-400 hover:underline" to="/builds">
                    {name}
                  </NavLink>
                  {repo_name && (
                    <>
                      <span> / </span>
                      <NavLink className="font-bold text-blue-400 hover:underline" to={`/${type}/${name}/repos/${repo_name}`}>
                        {repo_name}
                      </NavLink>
                    </>
                  )}
                  {build_name && <span> | Build #{build_name}</span>}
                </>
              )}
            </h1>
          </div>
          {/*TODO: Move legal entity select here <LegalEntitySelect />*/}
        </div>
        <div className="flex gap-x-6 text-gray-600">
          {navigationTabs.map((node) => (
            <div className={`border-b-2 ${node.active ? 'border-b-primary' : 'border-b-gray-100'}`} key={node.path}>
              <NavLink className={`flex cursor-pointer items-center gap-x-2 rounded py-1 px-3 hover:bg-gray-200 `} to={node.path}>
                {node.icon}
                <span className="text-sm">{node.label}</span>
              </NavLink>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
}
