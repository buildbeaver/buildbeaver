export interface IToast {
  colour: string;
  icon: JSX.Element;
  id: number;
  isHidden: boolean;
  message: string;
  order: number;
  title: string;
}
