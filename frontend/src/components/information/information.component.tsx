import React from 'react';
import { IoInformationCircleSharp } from 'react-icons/io5';

interface Props {
  text: string;
}

export function Information(props: Props): JSX.Element {
  const { text } = props;

  return (
    <div className="flex items-center gap-x-2 rounded-md bg-gray-100 p-2 text-sm text-gray-500">
      <IoInformationCircleSharp className="text-curiousBlue" size={28} />
      {text}
    </div>
  );
}
