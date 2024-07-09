import React from 'react';
import { Repos } from '../repos/repos.component';

interface Props {
  refreshSetupStatus: () => void;
}

export function SetUpRepos(props: Props): JSX.Element {
  const { refreshSetupStatus } = props;

  return (
    <>
      <p>
        Please enable the GitHub repositories you wish BuildBeaver to operate on below.
      </p>
      <br/>
      <Repos repoEnabled={refreshSetupStatus} />
    </>
  );
}
