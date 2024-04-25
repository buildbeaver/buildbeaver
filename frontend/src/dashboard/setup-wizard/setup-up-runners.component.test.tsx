import React from 'react';
import { fireEvent, render, screen } from '@testing-library/react';
import { SetUpRunners } from './set-up-runners.component';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { mockOrg } from '../../mocks/models/org.mock';
import { TickContext } from '../../contexts/tick/tick.context';
import { mockUseStaticResourceList } from '../../mocks/hooks/useStaticResourceList.mock';
import { ResourceKind } from '../../enums/resource-kind.enum';
import { IRunner } from '../../interfaces/runner.interface';
import { BrowserRouter } from 'react-router-dom';
import { SetupContext } from '../../contexts/setup/setup.context';

describe('SetUpRunners', () => {
  const renderSetUpRunners = (): void => {
    render(
      <BrowserRouter>
        <CurrentLegalEntityContext.Provider value={{ currentLegalEntity: mockOrg() }}>
          <TickContext.Provider value={{ tick: false, flip: jest.fn() }}>
            <SetupContext.Provider value={{ isInSetupContext: true, setupPath: '/setup', setupUrl: '' }}>
              <SetUpRunners refreshSetupStatus={jest.fn()} />
            </SetupContext.Provider>
          </TickContext.Provider>
        </CurrentLegalEntityContext.Provider>
      </BrowserRouter>
    );
  };

  describe('when the register button is clicked', () => {
    it('should open the register runner form', () => {
      mockUseStaticResourceList<IRunner>(ResourceKind.Runner);
      renderSetUpRunners();

      expect(screen.getByRole('button', { name: 'Register' })).toBeInTheDocument();
      expect(screen.getByText('No runners registered')).toBeInTheDocument();

      fireEvent.click(screen.getByRole('button', { name: 'Register' }));

      expect(screen.getByText('Register a new Runner')).toBeInTheDocument();
      expect(screen.getByRole('button', { name: 'Cancel' })).toBeInTheDocument();

      fireEvent.click(screen.getByRole('button', { name: 'Cancel' }));

      expect(screen.getByText('No runners registered')).toBeInTheDocument();
    });
  });
});
