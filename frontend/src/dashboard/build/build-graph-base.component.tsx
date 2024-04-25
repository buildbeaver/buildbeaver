import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { BuildGraph } from './build-graph.component';

export function BuildGraphBase(): JSX.Element {
  return (
    <Routes>
      <Route path="" element={<BuildGraph />} />
      <Route path="*" element={<Navigate to=".." relative="path" replace />} />
    </Routes>
  );
}
