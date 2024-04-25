import React, { useContext } from 'react';
import { NavLink, useLocation } from 'react-router-dom';
import { FaGithub } from 'react-icons/fa';
import { IRepo } from '../../interfaces/repo.interface';
import { useStaticResourceList } from '../../hooks/resources/resource-list.hook';
import { SelectedLegalEntityContext } from '../../contexts/selected-legal-entity/selected-legal-entity.context';
import { getTypeForLegalEntity, isOrg } from '../../utils/legal-entity.utils';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { PlaceholderMessage } from '../../components/placeholder-message/placeholder-message.component';
import { InternalLink } from '../../components/internal-link/internal-link.component';
import { Sidebar } from '../../components/sidebar/sidebar.component';
import { LegalEntitySelect } from '../legal-entity-select/legal-entity-select.component';
import { Button } from '../../components/button/button.component';

export function BuildsSidebar(): JSX.Element {
  const location = useLocation();
  const { selectedLegalEntity } = useContext(SelectedLegalEntityContext);
  const { loading, error, response } = useStaticResourceList<IRepo>({
    url: selectedLegalEntity.repo_search_url,
    query: {
      filters: [
        {
          field: 'enabled',
          operator: '=',
          value: `${true}`
        }
      ]
    }
  });

  const selectedLegalEntityType = getTypeForLegalEntity(selectedLegalEntity);
  const buttonLabel = isOrg(selectedLegalEntity) ? 'View Organization' : 'Your Home';
  const reposLink = `/${selectedLegalEntityType}/${selectedLegalEntity.name}/repos`;

  const repoListItems = (): JSX.Element => {
    if (loading) {
      return <SimpleContentLoader />;
    }

    if (error || !response) {
      return <PlaceholderMessage message="Failed to load repos" />;
    }

    if (response.results.length === 0) {
      return <PlaceholderMessage message="No repositories enabled" />;
    }

    return (
      <>
        {response.results.map((repo: IRepo) => {
          return (
            <div key={repo.id} className="rounded p-1 hover:bg-gray-100">
              <NavLink
                className={({ isActive }) => 'flex items-center gap-x-4' + (isActive ? ' text-gray-400' : '')}
                to={`/${selectedLegalEntityType}/${selectedLegalEntity.name}/repos/${repo.name}/builds`}
              >
                <span className="icon">
                  <FaGithub />
                </span>
                <span className="menu-item-label">{repo.name}</span>
              </NavLink>
            </div>
          );
        })}
        {response.next_url && (
          <div className="flex justify-center">
            <InternalLink label="View all" to={reposLink} />
          </div>
        )}
      </>
    );
  };

  const render = (): JSX.Element => {
    if (location.pathname !== '/builds' || !selectedLegalEntity) {
      return <></>;
    }

    return (
      <Sidebar>
        <div className="flex flex-col gap-y-4">
          <LegalEntitySelect />
          <NavLink to={reposLink}>
            <Button label={buttonLabel} size="full" />
          </NavLink>
          <div className="flex flex-col">{repoListItems()}</div>
        </div>
      </Sidebar>
    );
  };

  return render();
}
