import React from 'react';
import { fireEvent, render, RenderResult, screen } from '@testing-library/react';
import { Pagination } from './pagination.component';
import { IResourceResponse } from '../../services/responses/resource-response.interface';

interface RenderOptions {
  pageChanged?: (url: string) => void;
  resourceResponse?: IResourceResponse<any>;
}

const defaultRenderOptions: RenderOptions = {
  pageChanged: () => {},
  resourceResponse: {
    next_url: '',
    prev_url: '',
    results: []
  }
};

describe('Pagination', () => {
  const createPagination = (renderOptions?: RenderOptions): JSX.Element => {
    const { pageChanged, resourceResponse } = {
      ...defaultRenderOptions,
      ...renderOptions
    };

    return <Pagination pageChanged={pageChanged!} response={resourceResponse!} />;
  };

  const renderPagination = (renderOptions?: RenderOptions): RenderResult => {
    return render(createPagination(renderOptions));
  };

  it('should not render any buttons when next_url and prev_url are empty', () => {
    renderPagination();

    expect(screen.queryByText('Prev')).toBeNull();
    expect(screen.queryByText('Next')).toBeNull();
  });

  it('should render pagination buttons when next_url and prev_url are populated', () => {
    const pageChangedSpy = jest.fn();
    const { rerender } = renderPagination({
      pageChanged: pageChangedSpy,
      resourceResponse: {
        next_url: 'page-3',
        prev_url: 'page-1',
        results: []
      }
    });

    const prevButton = screen.getByRole('button', { name: 'Prev' });
    const nextButton = screen.getByRole('button', { name: 'Next' });

    expect(prevButton).toBeInTheDocument();
    expect(nextButton).toBeInTheDocument();

    fireEvent.click(nextButton);

    expect(prevButton).toBeDisabled();
    expect(nextButton).toBeDisabled();
    expect(pageChangedSpy).toHaveBeenCalledTimes(1);

    rerender(
      createPagination({
        pageChanged: pageChangedSpy,
        resourceResponse: {
          next_url: '',
          prev_url: 'page-2',
          results: []
        }
      })
    );

    expect(prevButton).toBeEnabled();
    expect(screen.queryByText('Next')).toBeNull();

    fireEvent.click(prevButton);
    expect(prevButton).toBeDisabled();
  });
});
