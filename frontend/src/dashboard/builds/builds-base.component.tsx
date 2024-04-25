import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { Builds } from './builds.component';
import { TickProvider } from '../../contexts/tick/tick.provider';

/**
 * Handles routing for views related to builds.
 */
export function BuildsBase(): JSX.Element {
  return (
    <TickProvider>
      <Routes>
        <Route path="/" element={<Builds />} />
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </TickProvider>
  );
}
