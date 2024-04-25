import { useEffect } from 'react';

/**
 * Hook that runs the given callback when a click is detected outside ref. Example usage:
 *
 * export function MyFunctionalComponent(props: Props): JSX.Element {
 *   const node = useRef(null);
 *
 *   const runOnClickOutside = (): void => {
 *     console.log('foo');
 *   };
 *
 *   useClickOutside(node, runOnClickOutside);
 *
 *   return <div ref={node}></div>;
 * }
 */
export function useClickOutside(ref: any, onClickOutside: Function, dependencies: any[] = []): void {
  useEffect(() => {
    function handleClickOutside(event: any) {
      if (ref.current && !ref.current.contains(event.target)) {
        onClickOutside();
      }
    }

    document.addEventListener('mousedown', handleClickOutside);

    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
    };
  }, [ref].concat(dependencies));
}
