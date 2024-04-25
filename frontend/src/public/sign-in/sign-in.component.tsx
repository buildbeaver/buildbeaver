import React, { useContext, useState } from 'react';
import { IoLogoGithub, IoFlaskOutline } from 'react-icons/io5';
import { Button } from '../../components/button/button.component';
import { RootContext } from '../../contexts/root/root.context';
import { isAuthenticated } from '../../utils/auth.utils';
import { Navigate } from 'react-router-dom';

export function SignIn(): JSX.Element {
  const [isLoading, setIsLoading] = useState(false);
  const rootDocument = useContext(RootContext);

  if (isAuthenticated()) {
    return <Navigate to={'/builds'} />;
  }

  const handleGitHubClick = (): void => {
    const githubAuthenticationUrl = rootDocument.github_authentication_url;
    const successUrl = encodeURIComponent(window.location.origin);
    const errorUrl = encodeURIComponent(window.location.href);

    setIsLoading(true);
    window.location.href = `${githubAuthenticationUrl}?success_url=${successUrl}&error_url=${errorUrl}`;
  };

  const render = (): JSX.Element => {
    return (
      <div className="flex flex-1 flex-col items-center justify-center bg-alabaster">
        <div className="flex h-[300px] w-[400px] flex-col gap-y-4 rounded-md border bg-white p-6 shadow-xl">
          <div className="flex items-center justify-between">
            <span className="text-xl">BuildBeaver</span>
            <IoFlaskOutline size={18} />
          </div>
          <hr />
          <p>Welcome to the BuildBeaver beta.</p>
          <p>
            Sign in with <b>GitHub</b>:
          </p>
          <div className="flex justify-center">
            <Button loading={isLoading} onClick={handleGitHubClick}>
              <IoLogoGithub size={24} />
            </Button>
          </div>
        </div>
      </div>
    );
  };

  return render();
}
