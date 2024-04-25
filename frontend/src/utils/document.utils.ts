/**
 * Returns true if the element currently being focused by the user is an input or text area.
 */
export function isActiveElementTextInput(): boolean {
  const activeElementTagName = document.activeElement?.tagName.toLowerCase();

  return activeElementTagName === 'input' || activeElementTagName === 'textarea';
}
