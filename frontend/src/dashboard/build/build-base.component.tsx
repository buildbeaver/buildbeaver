import React, { useEffect, useState } from 'react';
import { useBuildUrl } from '../../hooks/build-url/build-url.hook';
import { Status } from '../../enums/status.enum';
import { Navigate, Route, Routes, useLocation, useParams } from 'react-router-dom';
import { isSkeletonErroredBuild } from '../../utils/build.utils';
import { useBuild } from '../../hooks/build/build.hook';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { jobFQN } from '../../utils/job.utils';

interface Props {
  path: string;
  getElement: (
    buildGraph: IBuildGraph | undefined,
    buildGraphError: IStructuredError | undefined,
    selectedJobName: string | undefined,
    jobSelected: (jobName: string) => void
  ) => JSX.Element;
}

export function BuildBase(props: Props): JSX.Element {
  const { path, getElement } = props;
  const { job_name } = useParams(); // fully-qualified name for job
  const buildUrl = useBuildUrl();
  const { buildGraph, buildGraphError } = useBuild(buildUrl);
  const location = useLocation();
  const [selectedJobName, setSelectedJobName] = useState<string | undefined>();

  useEffect(() => {
    if (selectedJobName) {
      return;
    }

    // Note: if we have a skeleton build then there's no work for us to do here
    if (isSkeletonErroredBuild(buildGraph)) {
      setSelectedJobName('');
      return;
    }

    let jobToSelect = buildGraph?.jobs && buildGraph.jobs.length > 0 ? jobFQN(buildGraph.jobs[0].job) : undefined;

    if (job_name) {
      jobToSelect = job_name;
    } else if (buildGraph?.build.status === Status.Running) {
      const runningJob = buildGraph.jobs?.find((jGraph) => jGraph.job.status === Status.Running)?.job;
      if (runningJob) {
        jobToSelect = jobFQN(runningJob);
      }
    }
    setSelectedJobName(jobToSelect);
  }, [buildGraph?.jobs, buildGraph?.build.status, job_name, selectedJobName, location]);

  // Keeps the selected job in sync when the user navigates using the browser back button
  useEffect(() => {
    if (job_name !== selectedJobName) {
      setSelectedJobName(job_name);
    }
  }, [location]);

  // Auto select a job when none is specified
  if (!job_name && selectedJobName) {
    return (
      <Routes>
        <Route path="*" element={<Navigate to={selectedJobName} replace />} />
      </Routes>
    );
  }

  // Auto select a job when the provided job is not in the build graph
  if (
    buildGraph?.jobs &&
    buildGraph.jobs.length > 0 &&
    job_name &&
    !buildGraph?.jobs?.some((job) => jobFQN(job.job) === job_name)
  ) {
    return (
      <Routes>
        <Route
          path="*"
          element={<Navigate to={`../../${buildGraph.build.name}/${path}/${jobFQN(buildGraph.jobs[0].job)}`} replace />}
        />
      </Routes>
    );
  }

  return (
    <Routes>
      <Route path="" element={getElement(buildGraph, buildGraphError, selectedJobName, setSelectedJobName)} />
      <Route path="*" element={<Navigate to=".." relative="path" replace />} />
    </Routes>
  );
}
