import React from 'react';
import { Routes, Route } from 'react-router-dom';
import { Dashboard } from './dashboard/dashboard.component';
import { RootDocumentProvider } from './contexts/root/root-document.provider';
import { SignIn } from './public/sign-in/sign-in.component';
import './App.scss';
import { Toaster } from './components/toaster/toaster.component';
import { ToasterProvider } from './contexts/toaster/toaster.provider';
import { SignOut } from './public/sign-out/sign-out.component';
import { ErrorBoundary } from './components/error-boundary/error-boundary.component';

export function App(): JSX.Element {
  return (
    <ErrorBoundary>
      <ToasterProvider>
        <Toaster />
        <RootDocumentProvider>
          <Routes>
            <Route path="/sign-out" element={<SignOut />} />
            <Route path="/sign-in" element={<SignIn />} />
            <Route path="*" element={<Dashboard />} />
          </Routes>
        </RootDocumentProvider>
      </ToasterProvider>
    </ErrorBoundary>
  );
}
