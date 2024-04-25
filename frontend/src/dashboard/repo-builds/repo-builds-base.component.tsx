import React from 'react';
import { Navigate, Route, Routes } from 'react-router-dom';
import { NotFound } from '../../components/not-found/not-found.component';
import { RepoBuilds } from './repo-builds.component';
import { BuildLogBase } from '../build/build-log-base.component';
import { BuildArtifactsBase } from '../build/build-artifacts-base.component';
import { BuildGraphBase } from '../build/build-graph-base.component';

/**
 * Handles routing for views related to builds in the context of a repo.
 */
export function RepoBuildsBase(): JSX.Element {
  return (
    <Routes>
      <Route path=":build_name" element={<Navigate to="graph" relative="path" replace />} />
      <Route path=":build_name/graph/*" element={<BuildGraphBase />} />
      <Route path=":build_name/log/*" element={<BuildLogBase />} />
      <Route path=":build_name/log/:job_name/*" element={<BuildLogBase />} />
      <Route path=":build_name/artifacts/*" element={<BuildArtifactsBase />} />
      <Route path=":build_name/artifacts/:job_name/*" element={<BuildArtifactsBase />} />
      <Route path="/" element={<RepoBuilds />} />
      <Route path="*" element={<NotFound />} />
    </Routes>
  );
}
