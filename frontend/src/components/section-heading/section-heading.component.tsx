import React from 'react';

interface Props {
  text: string;
}

export function SectionHeading(props: Props): JSX.Element {
  const { text } = props;

  return (
    <div>
      <span className="text-lg">{text}</span>
      <hr className="mt-2" />
    </div>
  );
}
