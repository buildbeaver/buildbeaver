import React, { useRef, useState } from 'react';
import { useClickOutside } from '../../hooks/click-outside/click-outside.hook';

export interface IMenuItem {
  content: React.ReactNode;
  key: string;
  onClick?: () => unknown;
}

interface Props {
  children: React.ReactNode;
  items: IMenuItem[];
}

export function Menu(props: Props): JSX.Element {
  const [isOpen, setIsOpen] = useState(false);
  const node = useRef(null);

  const close = (): void => {
    setIsOpen(false);
  };

  const toggle = (): void => {
    setIsOpen(!isOpen);
  };

  useClickOutside(node, close);

  const itemClicked = (item: IMenuItem) => {
    item.onClick && item.onClick();
    close();
  };

  return (
    <div className="relative flex flex-col" ref={node}>
      <div className="cursor-pointer" onClick={toggle}>
        {props.children}
      </div>
      {isOpen && (
        <div className="absolute right-0 top-[30px] z-10 my-1 flex w-48 flex-col rounded-md border bg-white py-1 text-gray-600 shadow-md">
          {props.items.map((item: IMenuItem) => (
            <div
              className="flex w-full cursor-pointer rounded p-2 text-sm hover:bg-gray-100"
              key={item.key}
              onClick={() => itemClicked(item)}
            >
              {item.content}
            </div>
          ))}
        </div>
      )}
    </div>
  );
}
