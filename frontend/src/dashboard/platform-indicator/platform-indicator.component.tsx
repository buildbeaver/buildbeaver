import React from 'react';
import { IoLogoApple, IoLogoTux, IoLogoWindows } from 'react-icons/io5';

interface Props {
  runsOn?: string[];
  size?: number;
}

const defaultProps: Props = {
  runsOn: [],
  size: 22
};

export function PlatformIndicator(props: Props): JSX.Element {
  const { size, runsOn } = {
    ...defaultProps,
    ...props
  };

  const operatingSystems = ['linux', 'macos', 'windows'];
  const operatingSystem = runsOn?.find((part) => operatingSystems.includes(part));
  const title = `Platform: ${runsOn?.join(', ') ?? 'unknown'}`;

  switch (operatingSystem) {
    case 'linux':
      return <IoLogoTux size={size} title={title} />;
    case 'macos':
      return <IoLogoApple size={size} title={title} />;
    case 'windows':
      return <IoLogoWindows size={size} title={title} />;
    default:
      return <></>;
  }
}
