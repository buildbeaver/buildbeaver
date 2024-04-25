import React, { Fragment, useContext, useRef, useState } from 'react';
import { Dialog as HeadlessDialog, Transition } from '@headlessui/react';
import { Button } from '../button/button.component';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { IActionButton } from '../../interfaces/action-button.interface';

interface Props {
  actionButton: IActionButton;
  children: JSX.Element;
  icon?: JSX.Element;
  isOpen: boolean;
  setIsOpen: React.Dispatch<React.SetStateAction<boolean>>;
  title: string;
}

/**
 * Dialog component used for showing a generic dialog in the app.
 *
 * Note: Currently assumes a danger dialog until we need more types where we can specify a DialogType
 */
export function Dialog(props: Props): JSX.Element {
  const { actionButton, children, icon, title, isOpen, setIsOpen } = props;
  const { toastError } = useContext(ToasterContext);

  const [isLoading, setIsLoading] = useState(false);

  const cancelButtonRef = useRef(null);

  const onButtonClick = () => {
    setIsLoading(true);

    try {
      actionButton.clicked();
      setIsOpen(false);
    } catch (error) {
      toastError(error.serverError?.message);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <Transition.Root show={isOpen} as={Fragment}>
      <HeadlessDialog as="div" open={isOpen} className="relative z-30" initialFocus={cancelButtonRef} onClose={setIsOpen}>
        <Transition.Child
          as={Fragment}
          enter="ease-out duration-300"
          enterFrom="opacity-0"
          enterTo="opacity-100"
          leave="ease-in duration-200"
          leaveFrom="opacity-100"
          leaveTo="opacity-0"
        >
          <div className="fixed inset-0 bg-gray-500 bg-opacity-75 transition-opacity" />
        </Transition.Child>

        <div className="fixed inset-0 z-30 overflow-y-auto">
          <div className="flex min-h-full items-center justify-center p-0 text-center">
            <Transition.Child
              as={Fragment}
              enter="ease-out duration-300"
              enterFrom="opacity-0 translate-y-4 translate-y-0 scale-95"
              enterTo="opacity-100 translate-y-0 scale-100"
              leave="ease-in duration-200"
              leaveFrom="opacity-100 translate-y-0 scale-100"
              leaveTo="opacity-0 translate-y-4 translate-y-0 scale-95"
            >
              <HeadlessDialog.Panel className="relative my-8 mx-4 w-full max-w-lg transform overflow-hidden rounded-lg bg-white text-left shadow-xl transition-all">
                <div className="bg-white p-6 px-4 pt-5 pb-4">
                  <div className="mt-0">
                    <div className="flex justify-between">
                      <HeadlessDialog.Title as="h3" className="text-lg font-medium leading-6 text-gray-900">
                        {title}
                      </HeadlessDialog.Title>
                      {icon}
                    </div>
                    <hr className="my-4" />
                    {children}
                  </div>
                </div>
                <div className="flex flex-row-reverse gap-5 bg-gray-50 py-3 px-6">
                  <Button
                    type={actionButton.type}
                    label={actionButton.text}
                    loading={isLoading}
                    onClick={() => onButtonClick()}
                    size="regular"
                  />
                  <Button type="secondary" label="Cancel" loading={isLoading} onClick={() => setIsOpen(false)} size="regular" />
                </div>
              </HeadlessDialog.Panel>
            </Transition.Child>
          </div>
        </div>
      </HeadlessDialog>
    </Transition.Root>
  );
}
