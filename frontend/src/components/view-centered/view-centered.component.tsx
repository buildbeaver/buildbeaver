import React from 'react';
import Footer from '../footer/footer.component';
import { ContextualHeader } from '../contextual-header/contextual-header.component';
import { INavigationTab } from '../../interfaces/navigation-tab.interface';

interface Props {
  children: React.ReactNode;
  navigationTabs: INavigationTab[];
}

export function ViewCentered(props: Props): JSX.Element {
  const { children, navigationTabs } = props;

  return (
    <div className="flex min-w-0 grow flex-col">
      <ContextualHeader navigationTabs={navigationTabs} />
      <div className="flex flex-grow flex-col justify-between p-6">
        <div className="flex w-full justify-center">
          <div className="flex min-w-0 basis-[1000px] flex-col">{children}</div>
        </div>
        <Footer />
      </div>
    </div>
  );
}
