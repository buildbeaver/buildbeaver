import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { RepoSecretView } from './repo-secret-view.component';
import { RepoSecretEdit } from './repo-secret-edit.component';

export function RepoSecretBase(): JSX.Element {
  return (
    <Routes>
      <Route path="/" element={<RepoSecretView />} />
      <Route path="/edit" element={<RepoSecretEdit />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}
