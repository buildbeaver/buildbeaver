import React from 'react';
import { IToast } from '../../interfaces/toast.interface';

type Toast = (message: string, title?: string) => void;

interface Context {
  clearToast: (id: number) => void;
  toasts: IToast[];
  toastError: Toast;
  toastInfo: Toast;
  toastSuccess: Toast;
  toastWarn: Toast;
}

const MockToast: Toast = () => {};

export const ToasterContext = React.createContext<Context>({
  toasts: [],
  clearToast: () => {},
  toastError: MockToast,
  toastInfo: MockToast,
  toastSuccess: MockToast,
  toastWarn: MockToast
});
