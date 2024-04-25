import React from 'react';
import { render, screen } from '@testing-library/react';
import { PlaceholderMessage } from './placeholder-message.component';

describe('PlaceholderMessage', () => {
  const renderPlaceholderMessage = (message: string): void => {
    render(<PlaceholderMessage message={message} />);
  };

  it('should render a placeholder message', () => {
    renderPlaceholderMessage('No items found');

    expect(screen.getByText('No items found')).toBeInTheDocument();
  });
});
