import React from 'react';
import { render, screen } from '@testing-library/react';
import { isActiveElementTextInput } from './document.utils';

describe('isActiveElementTextInput', () => {
  it('should return true when focusing a text input', async () => {
    render(<input type="text" />);

    const input = screen.getByRole('textbox');

    input.focus();

    expect(isActiveElementTextInput()).toBeTruthy();
  });

  it('should return true when focusing a text area', async () => {
    render(<textarea />);

    const input = screen.getByRole('textbox');

    input.focus();

    expect(isActiveElementTextInput()).toBeTruthy();
  });

  it('should return false when not focusing an input or text area', async () => {
    render(<input type="text" />);
    expect(isActiveElementTextInput()).toBeFalsy();
  });
});
