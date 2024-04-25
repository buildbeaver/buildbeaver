import React from 'react';

interface Props {
  checked: boolean;
  id: string;
  label: string;
  onChange: Function;
}

export function Checkbox(props: Props): JSX.Element {
  const onChange = (): void => {
    props.onChange();
  };

  return (
    <div className="flex cursor-pointer items-center">
      <input
        id={props.id}
        type="checkbox"
        className="h-5 w-5 cursor-pointer rounded bg-gray-100 text-blue-700"
        checked={props.checked}
        onChange={onChange}
      />
      <label htmlFor={props.id} className="ml-3 cursor-pointer select-none">
        {props.label}
      </label>
    </div>
  );
}
