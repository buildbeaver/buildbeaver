import React, { useContext } from 'react';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { IoCloseSharp } from 'react-icons/io5';
import './toaster.component.scss';

export function Toaster(): JSX.Element {
  const { toasts, clearToast } = useContext(ToasterContext);

  /**
   * Offsets toasts so that they stack underneath each other, based on order.
   */
  const calculateTopPx = (order: number): string => {
    const toastHeight = 80;
    const toastGap = 10;
    const toasterPadding = 30;

    return `${order * (toastHeight + toastGap) + toasterPadding}px`;
  };

  return (
    <>
      {toasts.map((toast, index) => {
        if (toast.isHidden) {
          // Hidden toasts are not rendered on screen. Instead, we add a React fragment to maintain their position in
          // the component tree. Without this we would see the timeout animations jump around between toasts as older
          // toasts time out.
          return <React.Fragment key={index} />;
        }

        return (
          <div
            className="absolute right-[30px] z-50 flex h-[80px] w-[400px] flex-col justify-between rounded-md border-x border-t bg-white text-sm shadow-md"
            style={{ top: calculateTopPx(toast.order) }}
            key={index}
          >
            <div className="p-2.5">
              <div className="flex items-center justify-between">
                <div className="flex min-w-0">
                  {toast.icon}
                  <span className="... truncate whitespace-nowrap font-bold" title={toast.title}>
                    {toast.title}
                  </span>
                </div>
                <IoCloseSharp className="cursor-pointer text-gray-400" size={20} onClick={() => clearToast(toast.id)} />
              </div>
              <span className="ml-6 line-clamp-2" title={toast.message}>
                {toast.message}
              </span>
            </div>
            <div className="flex h-[4px] rounded-b-xl border-x border-b">
              <div className={`slide-out bg-${toast.colour} h-[3px] rounded-b-xl`}></div>
              <div className={`slide-in h-[3px] rounded-b-xl bg-white`}></div>
            </div>
          </div>
        );
      })}
    </>
  );
}
