import React from 'react';
import { Navigate } from 'react-router-dom';
import { isAuthenticated } from '../../utils/auth.utils';

export function RequireAuth(params: { children: JSX.Element }): JSX.Element {
  return isAuthenticated() ? params.children : <Navigate to="/sign-out" />;
}
