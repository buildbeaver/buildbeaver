import React, { useState } from 'react';
import { IRepo } from '../../interfaces/repo.interface';
import { useLiveResourceList } from '../../hooks/resources/resource-list.hook';
import { BuildsList } from '../builds-list/builds-list.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { Pagination } from '../../components/pagination/pagination.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';

interface Props {
  repo: IRepo;
}

export function RepoBuildsList(props: Props): JSX.Element {
  const { repo } = props;
  const [buildsUrl, setBuildsUrl] = useState(repo.builds_url);
  const { response, error, loading } = useLiveResourceList<IBuildGraph>({ url: buildsUrl });

  const pageChanged = (url: string): void => {
    setBuildsUrl(url);
  };

  return (
    <div className="flex flex-col gap-y-4">
      {loading ? <SimpleContentLoader rowHeight={40} /> : <BuildsList builds={response?.results} error={error} />}
      {response && <Pagination response={response} pageChanged={pageChanged} />}
    </div>
  );
}
