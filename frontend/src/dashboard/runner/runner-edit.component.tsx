import React, { FormEvent, useContext, useEffect, useState } from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { useRunnerUrl } from '../../hooks/runner-url/runner-url.hook';
import { useRunner } from '../../hooks/runner/runner.hook';
import { Button } from '../../components/button/button.component';
import { updateRunner } from '../../services/runners.service';
import { useForm } from 'react-hook-form';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { NotFound } from '../../components/not-found/not-found.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';

/**
 * The editable form values for a Runner
 */
type FormValues = {
  name: string;
};

/**
 * RunnerEdit provides the ability to edit an existing Runner
 */
export function RunnerEdit(): JSX.Element {
  const [isSaving, setIsSaving] = useState(false);
  const [name, setName] = useState('');
  const [enabled, setEnabled] = useState(true);

  const { toastError, toastSuccess } = useContext(ToasterContext);

  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    setValue: setFormValue,
    formState: { errors }
  } = useForm<FormValues>({ mode: 'onBlur' });

  // Gather the Runner from the current Url
  const runnerUrl = useRunnerUrl();
  const { runner, runnerError, runnerLoading } = useRunner(runnerUrl);

  useEffect(() => {
    if (runner) {
      setName(runner.name);
      setEnabled(runner.enabled);
      setFormValue('name', runner.name);
    }
  }, [runner]);

  if (runnerError) {
    return (
      <>
        <span className="text-lg">Edit Runner</span>
        <div className="my-5">
          <StructuredError error={runnerError} fallback="Failed to load runner" handleNotFound={true} />
        </div>
      </>
    );
  }

  if (runnerLoading) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={200} />;
  }

  if (!runner) {
    return <NotFound />;
  }

  // Handle the name input changing
  const nameChanged = (event: FormEvent<HTMLInputElement>): void => {
    const newName = event.currentTarget.value;
    setName(newName);
  };

  // Handle the enabled checkbox changing
  const enabledChanged = (event: FormEvent<HTMLInputElement>): void => {
    const newEnabled = event.currentTarget.checked;
    setEnabled(newEnabled);
  };

  // Submit the Runner modifications to our API
  const onSubmit = async () => {
    setIsSaving(true);
    await updateRunner(runner, { name, enabled })
      .then(() => {
        toastSuccess(`${name} has been updated`, 'Runner updated');
        navigate('..');
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to update Runner'));
      })
      .finally(() => {
        setIsSaving(false);
      });
  };

  return (
    <>
      <span className="text-lg">Edit Runner</span>
      <form className="my-5 grid grid-cols-6 gap-x-5">
        <hr className="col-span-6 mb-4 text-gray-100" />
        <label className="col-span-2 mt-2 text-gray-700" htmlFor="runner-name">
          Name
        </label>
        <div className="col-span-4 flex flex-col">
          <input
            {...register('name', {
              required: { value: true, message: 'Name is required' },
              maxLength: { value: 100, message: 'Name must not exceed 100 characters' },
              pattern: {
                value: /^[a-zA-Z0-9._-]{1,100}$/i,
                message: 'Name must only contain alphanumeric, dash or underscore characters'
              }
            })}
            aria-invalid={errors.name ? 'true' : 'false'}
            className="solid-input"
            id="runner-name"
            onChange={nameChanged}
            type="text"
            value={name}
          />
          {errors.name && (
            <p className="my-1 text-sm text-red-600" role="alert">
              {errors.name?.message}
            </p>
          )}
        </div>
        <hr className="col-span-6 my-4 text-gray-100" />
        <label className="col-span-2 mt-2 text-gray-700" htmlFor="runner-name">
          Enabled
        </label>
        <div className="col-span-4 flex flex-col self-end">
          <input id="runner-enabled" onChange={enabledChanged} type="checkbox" checked={enabled} />
          {errors.name && (
            <p className="my-1 text-sm text-red-600" role="alert">
              {errors.name?.message}
            </p>
          )}
        </div>
        <hr className="col-span-6 mt-4 text-gray-100" />
      </form>
      <div className="flex justify-end gap-x-5">
        <NavLink to={'..'}>
          <Button label="Cancel" type={'secondary'} />
        </NavLink>
        <Button label="Save" loading={isSaving} onClick={handleSubmit(onSubmit)} />
      </div>
    </>
  );
}
