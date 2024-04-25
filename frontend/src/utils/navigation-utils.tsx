import React from 'react';
import { INavigationTab } from '../interfaces/navigation-tab.interface';
import {
  IoArchiveOutline,
  IoCubeOutline,
  IoGitNetwork,
  IoGlasses,
  IoLayers,
  IoNewspaperOutline,
  IoPlayForwardOutline
} from 'react-icons/io5';
import { ILegalEntity } from '../interfaces/legal-entity.interface';
import { getTypeForLegalEntity } from './legal-entity.utils';

export enum BuildTab {
  Graph = 'graph',
  Log = 'log',
  Artifacts = 'artifacts'
}

export enum LegalEntityTab {
  Builds = 'builds',
  Repos = 'repos',
  Runners = 'runners',
  Settings = 'settings'
}

export enum RepoTab {
  Builds = 'builds',
  Secrets = 'secrets'
}

const iconSize = 18;

export function makeBuildNavigationTabs(
  active: BuildTab,
  legalEntity: ILegalEntity,
  repoName: string,
  buildName: string,
  selectedJobName?: string
): INavigationTab[] {
  const legalEntityType = getTypeForLegalEntity(legalEntity);
  const basePath = `/${legalEntityType}/${legalEntity.name}/repos/${repoName}/builds/${buildName}`;
  const selectedJobPart = selectedJobName ? `/${selectedJobName}` : '';

  return [
    {
      active: active === BuildTab.Graph,
      icon: <IoLayers size={iconSize} />,
      label: 'Graph',
      path: `${basePath}/graph`
    },
    {
      active: active === BuildTab.Log,
      icon: <IoNewspaperOutline size={iconSize} />,
      label: 'Log',
      path: `${basePath}/log${selectedJobPart}`
    },
    {
      active: active === BuildTab.Artifacts,
      icon: <IoArchiveOutline size={iconSize} />,
      label: 'Artifacts',
      path: `${basePath}/artifacts${selectedJobPart}`
    }
  ];
}

export function makeLegalEntityNavigationTabs(active: LegalEntityTab, legalEntity: ILegalEntity): INavigationTab[] {
  const legalEntityType = getTypeForLegalEntity(legalEntity);
  const basePath = `/${legalEntityType}/${legalEntity.name}`;

  return [
    {
      active: active === LegalEntityTab.Builds,
      icon: <IoCubeOutline size={iconSize} />,
      label: 'Builds',
      path: '/builds'
    },
    {
      active: active === LegalEntityTab.Repos,
      icon: <IoGitNetwork size={iconSize} />,
      label: 'Repos',
      path: `${basePath}/repos`
    },
    {
      active: active === LegalEntityTab.Runners,
      icon: <IoPlayForwardOutline size={iconSize} />,
      label: 'Runners',
      path: `${basePath}/runners`
    }
    // TODO: This is hidden until we have content to show at this route
    // {
    //   active: active === LegalEntityTab.Settings,
    //   icon: <IoSettingsOutline size={iconSize} />,
    //   label: 'Settings',
    //   path: `${basePath}/settings`
    // }
  ];
}

export function makeRepoNavigationTabs(active: RepoTab, legalEntity: ILegalEntity, repoName: string): INavigationTab[] {
  const legalEntityType = getTypeForLegalEntity(legalEntity);
  const basePath = `/${legalEntityType}/${legalEntity.name}/repos/${repoName}`;

  return [
    {
      active: active === RepoTab.Builds,
      icon: <IoCubeOutline size={iconSize} />,
      label: 'Builds',
      path: `${basePath}/builds`
    },
    {
      active: active === RepoTab.Secrets,
      icon: <IoGlasses size={iconSize} />,
      label: 'Secrets',
      path: `${basePath}/secrets`
    }
  ];
}
