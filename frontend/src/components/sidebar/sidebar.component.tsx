import React from 'react';

interface Props {
  children: React.ReactNode;
}

export function Sidebar(props: Props): JSX.Element {
  const { children } = props;

  return <div className="flex min-h-full w-[240px] flex-col">{children}</div>;
}
