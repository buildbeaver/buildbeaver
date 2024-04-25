import React from 'react';

interface Props {
  children: JSX.Element[];
}

export function List(props: Props): JSX.Element {
  return (
    <div className="rounded-md border shadow">
      {props.children.map((listItem, index) => {
        return (
          <div key={index}>
            {index > 0 && <hr />}
            {listItem}
          </div>
        );
      })}
    </div>
  );
}
