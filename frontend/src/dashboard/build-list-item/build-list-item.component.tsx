import React from 'react';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { NavLink } from 'react-router-dom';
import { IoCalendarClearOutline, IoStopwatchOutline } from 'react-icons/io5';
import { BuildStatusIndicator } from '../build-status-indicator/build-status-indicator.component';
import Gravatar from 'react-gravatar';
import { IAnyLegalEntity } from '../../interfaces/any-legal-entity.interface';
import { backgroundColour, buildDuration, createdAt, trimRef, trimSha } from '../../utils/build.utils';

interface Props {
  bGraph: IBuildGraph;
  isFirst: boolean;
  isLast: boolean;
  legalEntity: IAnyLegalEntity;
}

export function BuildListItem(props: Props): JSX.Element {
  const { bGraph, isFirst, isLast, legalEntity } = props;

  const borderRadiusStyle = (): string => {
    if (isFirst && isLast) {
      return 'rounded-md';
    }

    if (isFirst) {
      return 'rounded-t-md';
    }

    if (isLast) {
      return 'rounded-b-md';
    }

    return '';
  };

  const statusBorderRadiusStyle = (): string => {
    if (isFirst && isLast) {
      return 'rounded-l-md';
    }

    if (isFirst) {
      return 'rounded-tl-md';
    }
    if (isLast) {
      return 'rounded-bl-md';
    }
    return '';
  };

  return (
    <NavLink
      className={`flex cursor-pointer justify-between text-sm text-gray-600 hover:bg-gray-100 ${borderRadiusStyle()}`}
      to={`/${legalEntity.type}/${legalEntity.name}/repos/${bGraph.repo.name}/builds/${bGraph.build.name}`}
    >
      <div className="flex min-w-0 grow">
        <div className={`w-[4px] min-w-[4px] ${backgroundColour(bGraph.build.status)} ${statusBorderRadiusStyle()}`}></div>
        <div className="flex w-[40%] min-w-[40%] gap-x-2 p-3">
          <div className="flex flex-col items-center gap-y-0.5">
            <BuildStatusIndicator status={bGraph.build.status} size={20}></BuildStatusIndicator>
            <Gravatar email={bGraph.commit.author_email} className="h-[20px] w-[20px] rounded-full" />
          </div>
          <div className="flex min-w-0 flex-col gap-y-1">
            <div className="... truncate" title={`Build #${bGraph.build.name}`}>
              {legalEntity.name} / {bGraph.repo.name} <strong>#{bGraph.build.name}</strong>
            </div>
            <span className="text-xs">
              Committed by <strong>{bGraph.commit.author_name}</strong>
            </span>
          </div>
        </div>
        <div className="flex min-w-0 grow flex-col gap-y-1 p-3">
          <div className="... truncate" title="Commit message">
            {bGraph.commit.message}
          </div>
          <div className="text-xs">
            <span title="Branch">
              <strong>{trimRef(bGraph.build.ref)}</strong> /{' '}
            </span>
            <span title="Sha">{trimSha(bGraph.commit.sha)}</span>
          </div>
        </div>
      </div>
      <div className="flex w-[150px] min-w-[150px] flex-col justify-center gap-y-1 p-3 text-xs">
        <div className="flex items-center gap-x-2" title="Created">
          <IoCalendarClearOutline className="ml-[1px]" size={14} />
          {createdAt(bGraph.build.created_at)}
        </div>
        <div className="flex items-center gap-x-2" title="Duration">
          <IoStopwatchOutline size={16} />
          {buildDuration(bGraph.build.timings)}
        </div>
      </div>
    </NavLink>
  );
}
