import React from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { replacePathParts } from '../../utils/path.utils';

interface Props {
  children: JSX.Element[];
  selectedTab?: string;
  tabs: string[];
}

// TODO: This component now overlaps significantly with TabbedNavigation
export function TabbedContent(props: Props): JSX.Element {
  const { children, selectedTab, tabs } = props;
  const location = useLocation();

  const selectedTabIndexOrDefault = () => {
    const selectedTabIndex = tabs.findIndex((tab) => tab.toLowerCase() === selectedTab?.toLowerCase());

    if (selectedTabIndex === -1) {
      return 0;
    }

    return selectedTabIndex;
  };

  const selectedTabIndex = selectedTabIndexOrDefault();

  const borderClass = (index: number) => {
    return index === selectedTabIndex ? 'border-b-2 border-primary' : 'border-b';
  };

  const renderTab = (tab: string, index: number) => {
    const tabPath = replacePathParts(location.pathname, [{ positionFromEnd: 1, replacement: tab }]).toLowerCase();

    return (
      <NavLink className={`flex w-44 justify-center p-1 text-sm ${borderClass(index)}`} key={index} to={tabPath}>
        {tab}
      </NavLink>
    );
  };

  return (
    <div className="my-4 flex grow flex-col gap-y-4">
      <div className="flex w-full cursor-pointer">
        {tabs.map((tab, index) => renderTab(tab, index))}
        <div className="grow border-b" />
      </div>
      {children[selectedTabIndex]}
    </div>
  );
}
