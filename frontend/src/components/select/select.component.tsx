import React, { useRef, useState } from 'react';
import { FaChevronDown } from 'react-icons/fa';
import { useClickOutside } from '../../hooks/click-outside/click-outside.hook';

export interface ISelectItem {
  content: JSX.Element;
  label: string;
  onClick?: () => unknown;
}

interface Props {
  items: ISelectItem[];
  selectedItem: string;
}

export function Select(props: Props): JSX.Element {
  const { items, selectedItem } = props;
  const [isOpen, setIsOpen] = useState(false);
  const node = useRef(null);

  const close = (): void => {
    setIsOpen(false);
  };

  const toggle = (): void => {
    setIsOpen(!isOpen);
  };

  useClickOutside(node, close);

  const itemClicked = (item: ISelectItem): void => {
    if (item.label !== selectedItem && item.onClick) {
      item.onClick();
    }

    toggle();
  };

  return (
    <div className="relative flex flex-col text-sm text-gray-600" ref={node}>
      <div
        className="flex cursor-pointer items-center justify-between gap-x-2 rounded-md border bg-white p-2 shadow-md hover:bg-gray-100"
        onClick={toggle}
      >
        <span className="grow select-none truncate text-center">{selectedItem}</span>
        <FaChevronDown />
      </div>
      {isOpen && (
        <div className="absolute top-[40px] z-10 my-1 flex w-48 flex-col rounded-md border bg-white py-1 shadow-md">
          {items.map((item: ISelectItem) => (
            <div
              className={`flex w-full cursor-pointer hover:bg-gray-100 ${item.label === selectedItem ? 'bg-gray-100' : ''}`}
              key={item.label}
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
