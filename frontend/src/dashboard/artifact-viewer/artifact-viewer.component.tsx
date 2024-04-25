import React, { useEffect, useState } from 'react';
import { IArtifactDefinition } from '../../interfaces/artifact-definition.interface';
import { isFinished } from '../../utils/build-status.utils';
import { Loading } from '../../components/loading/loading.component';
import { IJob } from '../../interfaces/job.interface';
import { useStaticResourceList } from '../../hooks/resources/resource-list.hook';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { PlaceholderMessage } from '../../components/placeholder-message/placeholder-message.component';
import { IoMdDownload } from 'react-icons/io';
import prettyBytes from 'pretty-bytes';
import { Pagination } from '../../components/pagination/pagination.component';

interface Props {
  artifactSearchUri: string;
  selectedJob: IJob;
}

export function ArtifactViewer(props: Props): JSX.Element {
  const { artifactSearchUri, selectedJob } = props;
  const [artifactsUrl, setArtifactsUrl] = useState(artifactSearchUri);
  const {
    loading,
    error,
    response: artifacts,
    refresh: refreshArtifacts
  } = useStaticResourceList<IArtifactDefinition>({
    url: artifactsUrl,
    query: {
      workflow: selectedJob.workflow,
      job_name: selectedJob.name
    }
  });

  useEffect(() => {
    refreshArtifacts();
  }, [selectedJob.workflow, selectedJob.name, selectedJob.status]);

  if (loading) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={200} />;
  }

  if (error) {
    return <StructuredError error={error} fallback="Failed to load artifacts" />;
  }

  if (!isFinished(selectedJob.status)) {
    return <Loading message="Artifacts will become available after the job has completed..." />;
  }

  if (!artifacts || artifacts.results.length === 0) {
    return (
      <div className="p-3">
        <PlaceholderMessage message="No artifacts defined for this job" />
      </div>
    );
  }

  const pageChanged = (url: string): void => {
    setArtifactsUrl(url);
  };

  return (
    <>
      <table className="table-auto">
        <thead>
          <tr>
            <th>Name</th>
            <th>Path</th>
            <th>Size</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          {artifacts.results.map((artifact) => (
            <tr key={artifact.name}>
              <td>{artifact.group_name}</td>
              <td>{artifact.path}</td>
              <td className="whitespace-nowrap">{prettyBytes(artifact.size)}</td>
              <td>
                <a
                  className="flex items-center gap-x-2 text-blue-500 hover:text-gray-400"
                  title="Download artifact"
                  href={artifact.data_url}
                  download={artifact.name}
                >
                  <IoMdDownload size={18} />
                  Download
                </a>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <Pagination response={artifacts} pageChanged={pageChanged} />
    </>
  );
}
