import React from 'react';
import { NavLink } from 'react-router-dom';

interface Props {
  label: string;
  to: string;
}

/**
 * A wrapper around NavLink with common styling.
 */
export function InternalLink(props: Props): JSX.Element {
  const { label, to } = props;

  return (
    <NavLink to={to}>
      <span className="cursor-pointer text-gray-500 underline">{label}</span>
    </NavLink>
  );
}
