import React from 'react';
import { render, screen } from '@testing-library/react';
import { BrowserRouter } from 'react-router-dom';
import { Sidebar } from './sidebar.component';

describe('sidebar', () => {
  it('should render static sidebar content', () => {
    render(
      <BrowserRouter>
        <Sidebar>
          <div>Some sidebar content</div>
        </Sidebar>
      </BrowserRouter>
    );

    expect(screen.getByText('Some sidebar content')).toBeInTheDocument();
  });
});
