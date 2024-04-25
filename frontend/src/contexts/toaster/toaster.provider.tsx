import React, { useState } from 'react';
import { ToasterContext } from './toaster.context';
import { IToast } from '../../interfaces/toast.interface';
import { IoCheckmarkCircleSharp, IoCloseCircleSharp, IoInformationCircleSharp, IoWarning } from 'react-icons/io5';

/**
 * Provides toast notifications for users. Includes some trickery that lets us have toasts disappear after a set
 * interval whilst also showing a timeout animation on the toasts.
 */
export function ToasterProvider(props: any): JSX.Element {
  const [toasts, setToasts] = useState<IToast[]>([]);
  // When changing this value you must also change the css animation-duration in toaster.component.scss to match
  const toastDurationMs = 6000;

  const baseToast = (message: string, title: string): IToast => {
    return {
      id: generateId(),
      isHidden: false,
      message,
      order: -1,
      title
    } as IToast;
  };

  /**
   * Now in unix epoch ms. This is unique enough as we only pop a single toast at a time.
   */
  const generateId = (): number => {
    return Date.now();
  };

  /**
   * Timed out toasts are hidden instead of deleted. Without this the timeout animation would jump around as toasts
   * are removed from the toasts array.
   */
  const hideToast = (id: number) => {
    setToasts((toasts) =>
      toasts.map((toast) => {
        if (toast.id === id) {
          toast.isHidden = true;
        }

        return toast;
      })
    );
  };

  /**
   * Assigns the order that toasts will appear on screen. Hidden toasts are assigned -1 because they will not be
   * rendered anyway.
   */
  const orderVisibleToasts = (toasts: IToast[]): IToast[] => {
    let order = 0;

    return toasts.map((toast: IToast) => {
      if (toast.isHidden) {
        toast.order = -1;
      } else {
        toast.order = order;
        order++;
      }

      return toast;
    });
  };

  /**
   * Returns the array of toasts if any are visible. If they are all hidden returns and empty array which effectively
   * clears all in memory toasts from the toasts state. This lets us flush the state safely while no animations are on
   * screen.
   */
  const toastsOrEmpty = (toasts: IToast[]): IToast[] => {
    return toasts.every((toast) => toast.isHidden) ? [] : toasts;
  };

  /**
   * Adds a new toast to the toasts state and then hides it once it has timed out.
   */
  const setToast = (toast: IToast): void => {
    setToasts((toasts) => [...toastsOrEmpty(toasts), toast]);

    setTimeout(() => {
      hideToast(toast.id);
    }, toastDurationMs);
  };

  const toastError = (message: string, title = 'Error'): void => {
    const toast = {
      ...baseToast(message, title),
      colour: 'amaranth',
      icon: <IoCloseCircleSharp className={'mr-1.5 text-amaranth'} size={18} title={title} />
    };

    setToast(toast);
  };

  const toastInfo = (message: string, title = 'Info'): void => {
    const toast = {
      ...baseToast(message, title),
      colour: 'curiousBlue',
      icon: <IoInformationCircleSharp className={'mr-1.5 text-curiousBlue'} size={18} title={title} />
    };

    setToast(toast);
  };

  const toastSuccess = (message: string, title = 'Success'): void => {
    const toast = {
      ...baseToast(message, title),
      colour: 'mountainMeadow',
      icon: <IoCheckmarkCircleSharp className={'mr-1.5 text-mountainMeadow'} size={18} title={title} />
    };

    setToast(toast);
  };

  const toastWarn = (message: string, title = 'Warning'): void => {
    const toast = {
      ...baseToast(message, title),
      colour: 'flushOrange',
      icon: <IoWarning className={'mr-1.5 text-flushOrange'} size={18} title={title} />
    };

    setToast(toast);
  };

  const providerValue = {
    toasts: orderVisibleToasts(toasts),
    clearToast: hideToast,
    toastError,
    toastInfo,
    toastSuccess,
    toastWarn
  };

  return <ToasterContext.Provider value={providerValue}>{props.children}</ToasterContext.Provider>;
}
