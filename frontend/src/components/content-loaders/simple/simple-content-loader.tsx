import React from 'react';
import ContentLoader from 'react-content-loader';

interface Props {
  gapY?: number;
  numberOfRows?: number;
  rowHeight?: number;
}

export function SimpleContentLoader(props: Props): JSX.Element {
  const gapY = props.gapY ?? 10;
  const numberOfRows = props.numberOfRows ?? 4;
  const rowHeight = props.rowHeight ?? 20;

  const buildRows = (): JSX.Element[] => {
    return [...Array(numberOfRows)].map((_, index) => {
      const y = index * rowHeight + index * gapY;
      return <rect height={rowHeight} key={index} rx="4" ry="4" width="100%" y={y} />;
    });
  };

  const height = numberOfRows * rowHeight + (numberOfRows - 1) * gapY;

  return (
    <ContentLoader height={height} speed={1} width="100%">
      {buildRows()}
    </ContentLoader>
  );
}
