import React from 'react';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { List } from '../../components/list/list.component';
import { BuildListItem } from '../build-list-item/build-list-item.component';
import { useAnyLegalEntity } from '../../hooks/any-legal-entity/any-legal-entity.hook';
import { PlaceholderMessage } from '../../components/placeholder-message/placeholder-message.component';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { StructuredError } from '../../components/structured-error/structured-error.component';

interface Props {
  builds?: IBuildGraph[];
  error?: IStructuredError;
}

export function BuildsList(props: Props): JSX.Element {
  const { builds, error } = props;
  const legalEntity = useAnyLegalEntity();

  if (error) {
    return <StructuredError error={error} fallback="Failed to load builds" />;
  }

  if (!builds || builds.length === 0) {
    return <PlaceholderMessage message="No builds to display" />;
  }

  const isFirst = (index: number): boolean => {
    return index === 0;
  };

  const isLast = (index: number): boolean => {
    return index === builds.length - 1;
  };

  return (
    <List>
      {builds.map((bGraph: IBuildGraph, index: number) => (
        <BuildListItem key={index} bGraph={bGraph} isFirst={isFirst(index)} isLast={isLast(index)} legalEntity={legalEntity} />
      ))}
    </List>
  );
}
