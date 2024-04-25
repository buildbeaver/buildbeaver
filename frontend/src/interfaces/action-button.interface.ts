export interface IActionButton {
  text: string;
  type: 'primary' | 'secondary' | 'danger';
  clicked: () => unknown;
}
