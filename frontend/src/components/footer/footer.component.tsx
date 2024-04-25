import React from 'react';

class Footer extends React.Component<any, any> {
  render(): JSX.Element {
    return (
      <div className="flex w-full flex-col items-center justify-center gap-y-2 p-2">
        <div className="flex items-center justify-center gap-x-10 text-gray-600">
          <div>Home</div>
          <div>Docs</div>
          <div>Status</div>
          <div>Security</div>
        </div>
      </div>
    );
  }
}

export default Footer;
