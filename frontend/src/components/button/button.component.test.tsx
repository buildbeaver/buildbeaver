import React from 'react';
import { render, screen } from '@testing-library/react';
import { Button } from './button.component';

describe('Button', () => {
  it('should be disabled while loading', () => {
    render(<Button label="Foo" loading={true} />);

    expect(screen.getByRole('button')).toBeDisabled();
  });
});
