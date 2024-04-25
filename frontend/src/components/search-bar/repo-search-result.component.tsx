import React from 'react';
import { NavLink } from 'react-router-dom';
import { IRepo } from '../../interfaces/repo.interface';
import { useLegalEntityById } from '../../hooks/legal-entity-by-id/legal-entity-by-id.hook';
import { SimpleContentLoader } from '../content-loaders/simple/simple-content-loader';
import { getTypeForLegalEntity } from '../../utils/legal-entity.utils';
import { FailedSearchResult } from './failed-search-result.component';

interface Props {
  isFocused: boolean;
  isLast: boolean;
  repo: IRepo;
  clicked: () => void;
}

/**
 * Renders a single repo as a search result.
 */
export const RepoSearchResult = React.forwardRef<HTMLAnchorElement, Props>(
  (props: Props, ref: React.RefObject<HTMLAnchorElement>): JSX.Element => {
    const { isFocused, isLast, repo, clicked } = props;
    const { legalEntity, legalEntityError } = useLegalEntityById(repo.legal_entity_id);

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
        className={`flex cursor-pointer justify-between gap-x-4 p-2 pl-3 hover:bg-blue-100 ${
          isLast ? 'rounded-b-md' : 'border-b'
        } ${isFocused && 'bg-blue-200 hover:bg-blue-200'}`}
        ref={ref}
        to={`/${getTypeForLegalEntity(legalEntity)}/${legalEntity.name}/repos/${repo.name}`}
        onClick={clicked}
      >
        <div className="... flex-1 truncate" title="Repo name">
          {repo.name}
        </div>
        <div className="... flex-1 truncate text-right" title="Repo description">
          {repo.description === '' ? <span className="text-gray-300">No description available</span> : repo.description}
        </div>
      </NavLink>
    );
  }
);
