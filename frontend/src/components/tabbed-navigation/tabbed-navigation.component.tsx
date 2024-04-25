import React from 'react';
import { INavigationTab } from '../../interfaces/navigation-tab.interface';
import { NavLink, useLocation } from 'react-router-dom';

interface Props {
  tabs: INavigationTab[];
}

export function TabbedNavigation(props: Props): JSX.Element {
  const location = useLocation();
  const tabs = props.tabs;

  const toFullPath = (currentPath: string, tab: INavigationTab) => {
    const pathParts = currentPath.split('/');

    pathParts[pathParts.length - 1] = tab.path;

    return pathParts.join('/');
  };

  const borderClass = (currentPath: string, tab: INavigationTab) => {
    return location.pathname === toFullPath(currentPath, tab) ? 'border-b-2 border-primary' : 'border-b';
  };

  return (
    <div className="my-4 flex w-full cursor-pointer">
      {tabs.map((tab, index) => (
        <NavLink
          className={`flex w-44 justify-center p-1 text-sm ${borderClass(location.pathname, tab)}`}
          key={index}
          to={toFullPath(location.pathname, tab)}
        >
          {tab.label}
        </NavLink>
      ))}
      <div className="grow border-b" />
    </div>
  );
}
