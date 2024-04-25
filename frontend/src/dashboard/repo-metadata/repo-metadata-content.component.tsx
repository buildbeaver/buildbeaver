import React from 'react';
import { IRepo } from '../../interfaces/repo.interface';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { IoLogoGithub, IoGitBranch } from 'react-icons/io5';
import { RepoBuildStatus } from '../repo-build-status/repo-build-status.component';
import { Status } from '../../enums/status.enum';
import { useLiveResourceList } from '../../hooks/resources/resource-list.hook';
import { IBuildGraph } from '../../interfaces/build-graph.interface';

interface Props {
  repo: IRepo;
}

export function RepoMetadataContent(props: Props): JSX.Element {
  const { repo } = props;
  const {
    response: latestDefaultBuildResponse,
    error: latestDefaultBuildError,
    loading: latestDefaultBuildLoading
  } = useLiveResourceList<IBuildGraph>({
    url: `${repo.builds_url}?limit=1&q=ref:refs/heads/${repo.default_branch}`
  });
  const {
    response: latestBuildResponse,
    error: latestBuildError,
    loading: latestBuildLoading
  } = useLiveResourceList<IBuildGraph>({
    url: `${repo.builds_url}?limit=1&q=`
  });
  const latestDefaultBuild = latestDefaultBuildResponse?.results[0]?.build;
  const latestBuild = latestBuildResponse?.results[0]?.build;

  return (
    <div className="flex justify-between gap-x-6">
      <div className="flex min-w-0 flex-col gap-4 sm:flex-row">
        <div className="flex min-w-0 shrink-0 grow-0 flex-col whitespace-nowrap sm:max-w-[200px]">
          <div className="flex items-center" title={`Default branch: ${repo.default_branch}`}>
            <div className="mr-1">
              <IoGitBranch />
            </div>
            <span className="... truncate text-sm font-bold">{repo.default_branch}</span>
          </div>
          {latestDefaultBuild && <RepoBuildStatus status={latestDefaultBuild.status} />}
          {latestDefaultBuildResponse && !latestDefaultBuild && <span className="text-gray-400">-----</span>}
          {latestDefaultBuildLoading && (
            <div className="w-[50px]">
              <SimpleContentLoader numberOfRows={1} />
            </div>
          )}
          {latestDefaultBuildError && <RepoBuildStatus status={Status.Unknown} />}
        </div>
        <div className="flex flex-col" title={`Total builds for ${repo.name}`}>
          <span className="text-sm font-bold">Builds</span>
          {latestBuild && <span className="text-gray-400">{latestBuild?.name}</span>}
          {latestBuildResponse && !latestBuild && <span className="text-gray-400">0</span>}
          {latestBuildLoading && (
            <div className="w-10">
              <SimpleContentLoader numberOfRows={1} />
            </div>
          )}
          {latestBuildError && <span className="text-amaranth">-----</span>}
        </div>
        <div className="flex min-w-0 flex-col">
          <span className="text-sm font-bold" title={`Description for ${repo.name}`}>
            Description
          </span>
          <span className="... truncate text-gray-400" title={repo.description}>
            {repo.description || 'No description available'}
          </span>
        </div>
      </div>
      <div className="flex flex-col justify-center">
        <a target="_blank" href={repo.http_url} rel="noopener noreferrer">
          <IoLogoGithub size={30} />
        </a>
      </div>
    </div>
  );
}
