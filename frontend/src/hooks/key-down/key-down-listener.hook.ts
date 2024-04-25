import { useEffect } from 'react';

/**
 * Runs a callback when the keydown event is fired.
 * @param targetKey - Listen for key down events for this key.
 * @param onKeyDown - Run this callback when the key down event fires.
 * @param disableOnKeyDown - Return true here to skip running the onKeyDown() callback. Defaults to false.
 */
export function useKeyDownListener(targetKey: string, onKeyDown: () => void, disableOnKeyDown = () => false): void {
  useEffect(() => {
    const keyDownListener = (event: KeyboardEvent): void => {
      if (event.key === targetKey && !disableOnKeyDown()) {
        event.preventDefault();
        onKeyDown();
      }
    };

    window.addEventListener('keydown', keyDownListener);

    return () => {
      window.removeEventListener('keydown', keyDownListener);
    };
  }, [targetKey, onKeyDown]);
}
