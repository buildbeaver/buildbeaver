import React from 'react';

interface Props {
  children: React.ReactNode;
}

export function Centered(props: Props): JSX.Element {
  return (
    <div className="flex w-full justify-center">
      <div className="flex basis-[1000px] flex-col">{props.children}</div>
    </div>
  );
}
