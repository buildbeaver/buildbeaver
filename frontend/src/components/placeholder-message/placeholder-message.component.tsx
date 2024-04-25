import React from 'react';

interface Props {
  message: string;
}

export function PlaceholderMessage(props: Props): JSX.Element {
  const { message } = props;

  return <div className="flex justify-center p-2 text-gray-300">{message}</div>;
}
