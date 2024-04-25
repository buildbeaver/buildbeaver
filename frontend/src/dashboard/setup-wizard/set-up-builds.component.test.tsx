import React from 'react';
import { render, screen } from '@testing-library/react';
import { SetUpBuilds } from './set-up-builds.component';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { mockOrg } from '../../mocks/models/org.mock';
import { BrowserRouter } from 'react-router-dom';

describe('SetUpBuilds', () => {
  const renderSetUpBuilds = (): void => {
    render(
      <BrowserRouter>
        <CurrentLegalEntityContext.Provider value={{ currentLegalEntity: mockOrg() }}>
          <SetUpBuilds />
        </CurrentLegalEntityContext.Provider>
      </BrowserRouter>
    );
  };

  it('should render repos and runners links', () => {
    renderSetUpBuilds();

    expect(screen.getByText('Before continuing now is a good time to:')).toBeInTheDocument();

    const links = screen.queryAllByRole('link', { name: 'here' });

    expect(links).toHaveLength(2);
    expect(links[0]).toHaveAttribute('href', '/orgs/test-org/repos');
    expect(links[1]).toHaveAttribute('href', '/orgs/test-org/runners');
  });
});
