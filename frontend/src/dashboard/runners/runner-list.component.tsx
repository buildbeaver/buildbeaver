import React, { useContext, useEffect } from 'react';
import { IRunner } from '../../interfaces/runner.interface';
import { RunnerListItem } from './runner-list-item.component';
import { List } from '../../components/list/list.component';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { TickContext } from '../../contexts/tick/tick.context';
import { useStaticResourceList } from '../../hooks/resources/resource-list.hook';
import { PlaceholderMessage } from '../../components/placeholder-message/placeholder-message.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';

export function RunnerList(): JSX.Element {
  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { tick } = useContext(TickContext);
  const {
    loading,
    error,
    response: runners,
    refresh: refreshRunners
  } = useStaticResourceList<IRunner>({
    url: currentLegalEntity.runner_search_url,
    query: {}
  });

  useEffect(() => {
    refreshRunners();
  }, [tick]);

  if (loading) {
    return <SimpleContentLoader />;
  }

  if (error || !runners) {
    return <PlaceholderMessage message="Failed to load runners" />;
  }

  if (runners.results.length === 0) {
    return <PlaceholderMessage message="No runners registered" />;
  }

  const isFirst = (index: number): boolean => {
    return index === 0;
  };

  const isLast = (index: number): boolean => {
    return index === runners.results.length - 1;
  };

  return (
    <List>
      {runners.results.map((runner: IRunner, index: number) => (
        <RunnerListItem key={runner.id} isFirst={isFirst(index)} isLast={isLast(index)} runner={runner} />
      ))}
    </List>
  );
}
