import React from 'react';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { NavLink } from 'react-router-dom';
import { useLegalEntityById } from '../../hooks/legal-entity-by-id/legal-entity-by-id.hook';
import { SimpleContentLoader } from '../content-loaders/simple/simple-content-loader';
import { getTypeForLegalEntity } from '../../utils/legal-entity.utils';
import { backgroundColour, trimRef, trimSha } from '../../utils/build.utils';
import { FailedSearchResult } from './failed-search-result.component';

interface Props {
  bGraph: IBuildGraph;
  isFocused: boolean;
  isLast: boolean;
  clicked: () => void;
}

/**
 * Renders a single build as a search result.
 */
export const BuildSearchResult = React.forwardRef<HTMLAnchorElement, Props>(
  (props: Props, ref: React.RefObject<HTMLAnchorElement>): JSX.Element => {
    const { bGraph, isFocused, isLast, clicked } = props;
    const { legalEntity, legalEntityError } = useLegalEntityById(bGraph.repo.legal_entity_id);

    if (legalEntityError) {
      return <FailedSearchResult error={legalEntityError} isFocused={isFocused} isLast={isLast} />;
    }

    if (!legalEntity) {
      return (
        <div className="p-2">
          <SimpleContentLoader numberOfRows={1} />
        </div>
      );
    }

    return (
      <NavLink
        className={`cursor-pointer hover:bg-blue-100 ${isLast ? 'rounded-b-md' : 'border-b'} ${
          isFocused && 'bg-blue-200 hover:bg-blue-200'
        }`}
        ref={ref}
        to={`/${getTypeForLegalEntity(legalEntity)}/${legalEntity.name}/repos/${bGraph.repo.name}/builds/${bGraph.build.name}`}
        onClick={clicked}
      >
        <div className="flex">
          <div className={`w-[4px] min-w-[4px] ${backgroundColour(bGraph.build.status)}`}></div>
          <div className="flex min-w-0 flex-grow flex-col p-2">
            <div className="flex justify-between gap-x-4">
              <span className="... flex-1 truncate" title={`Build #${bGraph.build.name}`}>
                {legalEntity.name} / {bGraph.repo.name} <strong>#{bGraph.build.name}</strong>
              </span>
              <span className="... flex-1 truncate text-right" title="Commit message">
                {bGraph.commit.message}
              </span>
            </div>
            <div className="flex justify-between gap-x-4 text-xs">
              <span>
                Committed by <strong>{bGraph.commit.author_name}</strong>
              </span>
              <div>
                <span title="Branch">
                  <strong>{trimRef(bGraph.build.ref)}</strong> /{' '}
                </span>
                <span title="Sha">{trimSha(bGraph.commit.sha)}</span>
              </div>
            </div>
          </div>
        </div>
      </NavLink>
    );
  }
);
