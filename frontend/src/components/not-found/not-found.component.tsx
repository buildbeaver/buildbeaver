import React from 'react';
import { Button } from '../button/button.component';
import { NavLink } from 'react-router-dom';

export function NotFound(): JSX.Element {
  return (
    <div className="flex h-full flex-col items-center justify-center gap-y-3">
      <span className="text-xl font-bold text-primary">404</span>
      <span className="text-5xl font-bold">Page not found</span>
      <span>Sorry, we couldn't find the page you’re looking for.</span>
      <NavLink to="/">
        <Button label="Go back home" size="large" />
      </NavLink>
    </div>
  );
}
