import { BuildListItem } from './build-list-item.component';
import React from 'react';
import { BrowserRouter } from 'react-router-dom';
import { render, screen } from '@testing-library/react';
import { IAnyLegalEntity } from '../../interfaces/any-legal-entity.interface';
import { LegalEntityType } from '../../enums/legal-entity-type.enum';
import { ITimings } from '../../interfaces/timings.interface';
import { mockBuildGraph } from '../../mocks/models/build-graph.mock';

interface RenderOptions {
  timings?: ITimings;
}

const defaultRenderOptions: RenderOptions = {
  timings: undefined
};

describe('BuildListItem', () => {
  const mockOrg: IAnyLegalEntity = {
    name: 'buildbeaver',
    type: LegalEntityType.Orgs
  };

  const renderBuildListItem = (renderOptions?: RenderOptions): void => {
    const { timings } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    const build = mockBuildGraph({ timings });

    render(
      <BrowserRouter>
        <BuildListItem bGraph={build} isFirst={false} isLast={false} legalEntity={mockOrg} />
      </BrowserRouter>
    );
  };

  describe('successful build', () => {
    it('should render org, repo, and build information', () => {
      renderBuildListItem();

      expect(
        screen.getByText((content, node: Element) => node.textContent === 'buildbeaver / billys-playground #4')
      ).toBeInTheDocument();
    });

    it('should render committer information', () => {
      renderBuildListItem();

      expect(screen.getByText((content, node: Element) => node.textContent === 'Committed by Billy')).toBeInTheDocument();
    });

    it('should render the commit message', () => {
      renderBuildListItem();

      expect(screen.getByText('This is a test commit')).toBeInTheDocument();
    });

    it('should render the trimmed branch and commit sha', () => {
      renderBuildListItem();

      expect(screen.getByText((content, node: Element) => node.textContent === 'main / 6bdb713f0792')).toBeInTheDocument();
      expect(screen.queryByText('refs/heads/main')).toBeNull();
      expect(screen.queryByText('6bdb713f07928245a862b5e2bd3adc1c3c3c7346')).toBeNull();
    });

    it('should render a relative created time', () => {
      renderBuildListItem();

      expect(screen.getByText('1 day ago')).toBeInTheDocument();
    });

    it('should render the build duration', () => {
      renderBuildListItem({
        timings: {
          running_at: '2022-08-29T02:20:15.711825Z',
          finished_at: '2022-08-29T02:22:53.859466Z'
        }
      });

      expect(screen.getByText('02m 38s')).toBeInTheDocument();
    });
  });

  describe('queued build', () => {
    it('should render "Queued" instead of a build duration', () => {
      renderBuildListItem({
        timings: {
          queued_at: '2022-08-29T02:13:07.890265Z'
        }
      });

      expect(screen.getByText('Queued')).toBeInTheDocument();
    });
  });

  describe('submitted build', () => {
    it('should render "Submitted" instead of a build duration', () => {
      renderBuildListItem({
        timings: {
          queued_at: '2022-08-29T02:13:07.890265Z',
          submitted_at: '2022-08-29T02:20:15.594971Z'
        }
      });

      expect(screen.getByText('Submitted')).toBeInTheDocument();
    });
  });
});
