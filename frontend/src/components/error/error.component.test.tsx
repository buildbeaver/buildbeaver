import React from 'react';
import { render, screen } from '@testing-library/react';
import { Error } from './error.component';

describe('Error', () => {
  const renderError = (errorMessage: string): void => {
    render(<Error errorMessage={errorMessage} />);
  };

  it('should render an error message', () => {
    renderError('Creation of thing failed');

    expect(screen.getByText('Creation of thing failed')).toBeInTheDocument();
  });
});
