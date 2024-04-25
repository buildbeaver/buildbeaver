import React from 'react';
import { RunnerList } from './runner-list.component';
import { Button } from '../../components/button/button.component';
import { NavLink } from 'react-router-dom';

export function RunnersView(): JSX.Element {
  return (
    <>
      <div className="flex justify-end">
        <NavLink to="register">
          <Button label="Register" />
        </NavLink>
      </div>
      <div className="my-6 flex flex-col gap-y-4">
        <RunnerList />
      </div>
    </>
  );
}
