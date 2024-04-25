import React, { useContext, useState } from 'react';
import { RepoList } from './repo-list.component';
import { SectionHeading } from '../../components/section-heading/section-heading.component';
import { DateTime } from 'luxon';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';

interface Props {
  repoEnabled?: () => void;
}

export function Repos(props: Props): JSX.Element {
  const { repoEnabled } = props;
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const [lastRepoUpdated, setLastRepoUpdated] = useState<string>();

  /**
   * Every time a repo is updated we track it here, so that both lists will trigger a refresh regardless of which list
   * the update originated from. We need to use a unique-ish string to trigger the refresh in case multiple async
   * updates complete at a similar time. This will circumvent the React state update batching that would occur if we
   * used something simple like flipping a boolean instead.
   */
  const repoUpdated = (repoName: string, enabled: boolean): void => {
    // Append an additional timestamp in case the user enables / disables the same repo more than once in a row
    setLastRepoUpdated(`${repoName}_${DateTime.now().toMillis()}`);
    enabled && repoEnabled && repoEnabled();
  };

  const render = (): JSX.Element => {
    return (
      <div className="flex flex-col gap-y-6">
        <SectionHeading text="Enabled repos" />
        <RepoList
          enabled={true}
          lastRepoUpdated={lastRepoUpdated}
          repoSearchUrl={currentLegalEntity.repo_search_url}
          repoUpdated={repoUpdated}
        />
        <SectionHeading text="Available repos" />
        <RepoList
          enabled={false}
          lastRepoUpdated={lastRepoUpdated}
          repoSearchUrl={currentLegalEntity.repo_search_url}
          repoUpdated={repoUpdated}
        />
      </div>
    );
  };

  return render();
}
