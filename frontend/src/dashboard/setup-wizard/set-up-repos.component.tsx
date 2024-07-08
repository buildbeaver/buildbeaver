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
        In order for BuildBeaver to be able to operate against your GitHub repository, you must enable each repository below.
      </p>
      <Repos repoEnabled={refreshSetupStatus} />
    </>
  );
}
