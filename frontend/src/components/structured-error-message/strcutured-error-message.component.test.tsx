import React from 'react';
import { render, screen } from '@testing-library/react';
import { StructuredErrorMessage } from './structured-error-message.component';
import { IStructuredError } from '../../interfaces/structured-error.interface';

describe('StructuredErrorMessage', () => {
  const renderStructuredError = (error: IStructuredError, fallback?: string): void => {
    render(<StructuredErrorMessage error={error} fallback={fallback} />);
  };

  it('should return a server error message', () => {
    const error: IStructuredError = {
      serverError: {
        code: '1234',
        details: undefined,
        http_status_code: 400,
        message: 'Validation failed'
      },
      statusCode: 400,
      statusText: 'Bad request'
    };

    renderStructuredError(error, 'Creation of thing failed');

    expect(screen.getByText('Validation failed')).toBeInTheDocument();
  });

  it('should return a specified fallback message', () => {
    const error: IStructuredError = {
      statusCode: 400,
      statusText: 'Bad request'
    };

    renderStructuredError(error, 'Creation of thing failed');

    expect(screen.getByText('Creation of thing failed')).toBeInTheDocument();
  });

  it('should return the default fallback message', () => {
    const error: IStructuredError = {
      statusCode: 400,
      statusText: 'Bad request'
    };

    renderStructuredError(error);

    expect(screen.getByText('Something went wrong')).toBeInTheDocument();
  });
});
