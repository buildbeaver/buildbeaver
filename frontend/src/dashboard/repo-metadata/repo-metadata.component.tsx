import React from 'react';
import { IRepo } from '../../interfaces/repo.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { RepoMetadataContent } from './repo-metadata-content.component';

interface Props {
  repo?: IRepo;
  repoError?: IStructuredError;
}

export function RepoMetadata(props: Props): JSX.Element {
  const { repo, repoError } = props;

  if (repo) {
    return <RepoMetadataContent repo={repo} />;
  }

  if (repoError) {
    return <></>;
  }

  return <SimpleContentLoader numberOfRows={2} rowHeight={33} />;
}
