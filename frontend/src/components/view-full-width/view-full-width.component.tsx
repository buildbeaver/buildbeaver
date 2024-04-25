import React from 'react';
import Footer from '../footer/footer.component';
import { ContextualHeader } from '../contextual-header/contextual-header.component';
import { INavigationTab } from '../../interfaces/navigation-tab.interface';

interface Props {
  children: React.ReactNode;
  navigationTabs: INavigationTab[];
}

export function ViewFullWidth(props: Props): JSX.Element {
  const { children, navigationTabs } = props;

  return (
    <div className="flex grow flex-col">
      <ContextualHeader navigationTabs={navigationTabs} />
      <div className="flex flex-grow flex-col justify-between p-6">
        {children}
        <Footer />
      </div>
    </div>
  );
}
