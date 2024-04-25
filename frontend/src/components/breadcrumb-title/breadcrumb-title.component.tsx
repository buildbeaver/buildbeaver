import React from 'react';
import { NavLink } from 'react-router-dom';
import { useBreadcrumbs } from '../../hooks/breadcrumbs/breadcrumbs.hook';
import { IoHomeSharp, IoChevronForwardSharp } from 'react-icons/io5';

export function BreadcrumbTitle(): JSX.Element {
  const breadcrumbs = useBreadcrumbs();
  const hasMultipleCrumbs = breadcrumbs.length > 1;

  /**
   * Truncates long breadcrumb labels by removing the middle of the label and replacing it with ellipsis.
   * e.g. the-quick-brown-fox-jumps-over-the-lazy-dog truncates to the-quick-brown...er-the-lazy-dog. Fixes issues
   * with excessively long breadcrumbs e.g. long repo names or job names.
   */
  const truncate = (label: string): string => {
    if (label.length <= 30) {
      return label;
    }

    const start = label.slice(0, 15);
    const end = label.slice(label.length - 15, label.length);

    return `${start}...${end}`;
  };

  return (
    <div className="flex flex-wrap items-center text-gray-600">
      <NavLink className="pr-3" to="/">
        <IoHomeSharp className="cursor-pointer hover:text-gray-400" />
      </NavLink>
      {breadcrumbs && (
        <div>
          <IoChevronForwardSharp />
        </div>
      )}
      {breadcrumbs.map((crumb, index) => (
        <React.Fragment key={crumb.path}>
          <NavLink
            className="cursor-pointer whitespace-nowrap px-3 text-sm hover:text-gray-400"
            title={crumb.label}
            to={crumb.path}
          >
            {truncate(crumb.label)}
          </NavLink>
          {hasMultipleCrumbs && index < breadcrumbs.length - 1 && (
            <div>
              <IoChevronForwardSharp />
            </div>
          )}
        </React.Fragment>
      ))}
    </div>
  );
}
