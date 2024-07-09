import React, { useState } from 'react';
import { Button } from '../../components/button/button.component';
import { RunnerList } from '../runners/runner-list.component';
import { IoAddCircleOutline } from 'react-icons/io5';
import { RunnerRegister } from '../runner/runner-register.component';

enum Views {
  List = 'list',
  Register = 'register'
}

interface Props {
  refreshSetupStatus: () => void;
}

export function SetUpRunners(props: Props): JSX.Element {
  const { refreshSetupStatus } = props;
  const [view, setView] = useState(Views.List);

  const cancelClicked = (): void => {
    setView(Views.List);
  };

  const runnerRegistered = (): void => {
    setView(Views.List);
    refreshSetupStatus();
  };

  if (view === Views.Register) {
    return <RunnerRegister cancelClicked={cancelClicked} runnerRegistered={runnerRegistered} />;
  }

  return (
    <>
      <p>
        In order for BuildBeaver to be able to build your solution, you need to register runners via the button below.
      </p>
      <div className="flex justify-end">
        <Button label="Register" onClick={() => setView(Views.Register)}>
          <div className="ml-1">
            <IoAddCircleOutline size={22} />
          </div>
        </Button>
      </div>
      <div className="my-10 flex flex-col gap-y-4">
        <RunnerList />
      </div>
    </>
  );
}
