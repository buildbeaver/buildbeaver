import React from 'react';
import {NavLink} from "react-router-dom";

export function SetupComplete(): JSX.Element {
  return (
    <div>
      <p>
        Congratulations! Your BuildBeaver setup is now complete.
      </p>
      <br/>
      <p>

      </p>
      <p>We recommend reading the following guides to make the most out of your BuildBeaver experience:</p>
      <br/>
      <ol className="list-inside list-decimal">
        <li>
          <a className="text-blue-400" href="https://buildbeaver.github.io/docs/category/getting-started---go" target="_blank" rel="noreferrer">
            Getting Started - Go
          </a>
          .
        </li>
        <li>
          <a className="text-blue-400" href="https://buildbeaver.github.io/docs/category/guide-to-dynamic-builds" target="_blank" rel="noreferrer">
            Guide to Dynamic Builds
          </a>
          .
        </li>
      </ol>
    </div>
  );
}
