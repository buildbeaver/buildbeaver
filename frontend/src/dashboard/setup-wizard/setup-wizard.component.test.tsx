import React from 'react';
import { fireEvent, render, RenderResult, screen } from '@testing-library/react';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { mockOrg } from '../../mocks/models/org.mock';
import { BrowserRouter } from 'react-router-dom';
import { mockUseSetupStatus } from '../../mocks/hooks/useSetupStatus.mock';
import { IUseSetupStatus } from '../../hooks/setup-status/setup-status.hook';
import { SetupContext } from '../../contexts/setup/setup.context';
import { SetupWizard } from './setup-wizard.component';
import { ISetupStatus } from '../../interfaces/setup-status.interface';

describe('SetupWizard', () => {
  const createSetupWizard = (setupStatus: ISetupStatus): JSX.Element => {
    mockUseSetupStatus({ setupStatus } as IUseSetupStatus);

    return (
      <BrowserRouter>
        <CurrentLegalEntityContext.Provider value={{ currentLegalEntity: mockOrg() }}>
          <SetupContext.Provider value={{ isInSetupContext: true, setupPath: '', setupUrl: '' }}>
            <SetupWizard />
          </SetupContext.Provider>
        </CurrentLegalEntityContext.Provider>
      </BrowserRouter>
    );
  };

  const renderSetupWizard = (setupStatus: ISetupStatus): RenderResult => {
    return render(createSetupWizard(setupStatus));
  };

  it('should guide the user through setup steps', () => {
    const { rerender } = renderSetupWizard({
      builds_run: false,
      buildbeaver_installed: false,
      repos_enabled: false,
      runners_registered: false
    } as ISetupStatus);

    // Step 1
    expect(screen.getByText('Install the GitHub app')).toBeInTheDocument();
    expect(screen.getByText('Action required')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    rerender(
      createSetupWizard({
        builds_run: false,
        buildbeaver_installed: true,
        repos_enabled: false,
        runners_registered: false
      } as ISetupStatus)
    );

    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    // Step 2
    expect(screen.getByText('Enable repos')).toBeInTheDocument();
    expect(screen.getByText('Action required')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    rerender(
      createSetupWizard({
        builds_run: false,
        buildbeaver_installed: true,
        repos_enabled: true,
        runners_registered: false
      } as ISetupStatus)
    );

    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    // Step 3
    expect(screen.getByText('Register runners')).toBeInTheDocument();
    expect(screen.getByText('Action required')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    rerender(
      createSetupWizard({
        builds_run: false,
        buildbeaver_installed: true,
        repos_enabled: true,
        runners_registered: true
      } as ISetupStatus)
    );

    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    // Step 4
    expect(screen.getByText('Start building')).toBeInTheDocument();
    expect(screen.getByText('Action required')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeDisabled();

    rerender(
      createSetupWizard({
        builds_run: true,
        buildbeaver_installed: true,
        repos_enabled: true,
        runners_registered: true
      } as ISetupStatus)
    );

    expect(screen.getByText('Complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Next' })).toBeEnabled();

    fireEvent.click(screen.getByRole('button', { name: 'Next' }));

    // Complete
    expect(screen.getByText('Setup complete')).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Finish' })).toBeInTheDocument();
    expect(screen.getByRole('button', { name: 'Finish' })).toBeEnabled();
  });
});
