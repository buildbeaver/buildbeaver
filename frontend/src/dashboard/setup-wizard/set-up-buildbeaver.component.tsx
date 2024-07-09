import React from 'react';

export function SetUpBuildBeaver(): JSX.Element {
  return (
    <div>
      <p>
        In order for a BuildBeaver Server to be able to build the code from GitHub repos, a GitHub app is used to
        provide access rights to the server. By installing the GitHub app for a user account or repo a developer can
        enable access to one or more repos.
      </p>
      <br/>
      <p>
        Please follow the{' '}
        <a className="text-blue-400"
                 href="https://github.com/buildbeaver/buildbeaver/blob/main/creating-github-app.md" target="_blank" rel="noreferrer">
          Creating a BuildBeaver GitHub App
        </a>
        {' '} guide to continue.
      </p>
    </div>
  );
}
