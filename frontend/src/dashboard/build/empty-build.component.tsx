import React from 'react';
import { ViewCentered } from '../../components/view-centered/view-centered.component';
import { BuildMetadata } from '../build-metadata/build-metadata.component';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { INavigationTab } from '../../interfaces/navigation-tab.interface';

interface Props {
  buildGraph: IBuildGraph;
  navigationTabs: INavigationTab[];
}

export function EmptyBuild(props: Props): JSX.Element {
  const { buildGraph, navigationTabs } = props;

  return (
    <ViewCentered navigationTabs={navigationTabs}>
      <BuildMetadata buildGraph={buildGraph} />
      <p className="my-4">
        Lorem ipsum dolor sit amet, consectetur adipiscing elit. Nam finibus nunc in leo pulvinar tristique. Donec purus ligula,
        egestas accumsan est sed, maximus vehicula odio. Aliquam placerat sollicitudin augue. Donec vel vulputate mi. Nulla luctus
        neque sed mi sollicitudin aliquam. Donec vel justo quis dui dapibus hendrerit nec eget ipsum. Pellentesque mauris purus,
        ullamcorper id tristique non, mollis quis odio. Sed id molestie felis, eget rutrum ante. Duis pulvinar lacus purus. Fusce
        gravida egestas tellus sit amet fermentum. Aenean nec placerat nulla.
      </p>
    </ViewCentered>
  );
}
