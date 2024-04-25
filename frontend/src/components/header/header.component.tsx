import React, { useContext } from 'react';
import { IoMenuSharp } from 'react-icons/io5';
import { NavLink, useNavigate } from 'react-router-dom';
import { IMenuItem, Menu } from '../menu/menu.component';
import { RootContext } from '../../contexts/root/root.context';
import { SearchBar } from '../search-bar/search-bar.component';

export function Header(): JSX.Element {
  const navigate = useNavigate();
  const rootDocument = useContext(RootContext);

  const menuItems: IMenuItem[] = [
    {
      content: <span>Sign out</span>,
      key: 'sign-out',
      onClick: () => navigate('/sign-out')
    }
  ];

  const render = (): JSX.Element => {
    if (!rootDocument.current_legal_entity_url) {
      return <></>;
    }

    return (
      <div className="header flex items-center justify-between bg-primary px-4 py-2">
        <div className="flex-1 ">
          <NavLink className="cursor-pointer text-xl text-alabaster" to="/">
            BuildBeaver
          </NavLink>
        </div>
        <div className="flex-2">
          <SearchBar />
        </div>
        <div className="flex flex-1 items-center justify-end">
          <Menu items={menuItems}>
            <IoMenuSharp className="text-alabaster" size={26} />
          </Menu>
        </div>
      </div>
    );
  };

  return render();
}
