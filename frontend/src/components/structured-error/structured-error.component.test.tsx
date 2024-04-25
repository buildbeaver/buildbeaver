import React from 'react';
import { render, screen } from '@testing-library/react';
import { StructuredError } from './structured-error.component';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { BrowserRouter } from 'react-router-dom';

interface RenderOptions {
  error?: IStructuredError;
  fallback?: string;
  handleNotFound?: boolean;
}

const defaultError: IStructuredError = {
  serverError: {
    code: 'fatal',
    details: '',
    message: 'Something went horribly wrong',
    http_status_code: 500
  },
  statusCode: 500,
  statusText: 'An internal server error occurred'
};

const defaultRenderOptions: RenderOptions = {
  error: defaultError
};

describe('StructuredError', () => {
  const renderStructuredError = (renderOptions?: RenderOptions): void => {
    const { error, fallback, handleNotFound } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    render(
      <BrowserRouter>
        <StructuredError error={error!} fallback={fallback} handleNotFound={handleNotFound} />
      </BrowserRouter>
    );
  };

  it('should render a server error message', () => {
    renderStructuredError();

    expect(screen.getByText('Something went horribly wrong')).toBeInTheDocument();
  });

  it('should render a fallback message', () => {
    renderStructuredError({ error: { ...defaultError, serverError: undefined }, fallback: 'Meaningful fallback message' });

    expect(screen.getByText('Meaningful fallback message')).toBeInTheDocument();
  });

  it('should render a 404 message', () => {
    renderStructuredError({ error: { ...defaultError, statusCode: 404 }, handleNotFound: true });

    expect(screen.getByText('404')).toBeInTheDocument();
    expect(screen.getByText('Page not found')).toBeInTheDocument();
    expect(screen.getByText("Sorry, we couldn't find the page you’re looking for.")).toBeInTheDocument();
  });

  // Cannot find any info on how to test a <Navigate /> component
  it.todo('should redirect on 401');
});
