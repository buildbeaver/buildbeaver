import React from 'react';
import { Navigate } from 'react-router-dom';
import { clearSensitiveData } from '../../utils/auth.utils';

export function SignOut(): JSX.Element {
  clearSensitiveData();

  return <Navigate to={'/sign-in'} />;
}
