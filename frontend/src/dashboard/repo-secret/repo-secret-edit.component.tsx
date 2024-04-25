import React, { FormEvent, useContext, useEffect, useState } from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { Button } from '../../components/button/button.component';
import { useForm } from 'react-hook-form';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { useSecretUrl } from '../../hooks/secret-url/secret-url.hook';
import { updateSecret } from '../../services/secrets.service';
import { useSecret } from '../../hooks/secret/secret.hook';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { NotFound } from '../../components/not-found/not-found.component';

/**
 * The editable form values for a Secret
 */
type FormValues = {
  name: string;
  value: string;
};

/**
 * RepoSecretEdit provides the ability to edit an existing Secret
 */
export function RepoSecretEdit(): JSX.Element {
  const [isSaving, setIsSaving] = useState(false);
  const [name, setName] = useState('');
  const [value, setValue] = useState('');

  const { toastError, toastSuccess } = useContext(ToasterContext);

  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    setValue: setFormValue,
    formState: { errors }
  } = useForm<FormValues>({ mode: 'onBlur' });

  // Gather the Secret from the current Url
  const secretUrl = useSecretUrl();
  const { secret, secretError, secretLoading } = useSecret(secretUrl);

  useEffect(() => {
    if (secret) {
      setName(secret.name);
      setFormValue('name', secret.name);
    }
  }, [secret]);

  if (secretError) {
    return (
      <>
        <span className="text-lg">Edit Runner</span>
        <div className="my-5">
          <StructuredError error={secretError} fallback="Failed to fetch secret" handleNotFound={true} />
        </div>
      </>
    );
  }

  if (secretLoading) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={200} />;
  }

  if (!secret) {
    return <NotFound />;
  }

  // Handle the name input changing
  const nameChanged = (event: FormEvent<HTMLInputElement>): void => {
    const newName = event.currentTarget.value;
    setName(newName);
  };

  // Handle the value input changing
  const valueChanged = (event: FormEvent<HTMLInputElement>): void => {
    const newValue = event.currentTarget.value;

    setValue(newValue);
  };

  // Submit the Secret modifications to our API
  const onSubmit = async () => {
    setIsSaving(true);
    await updateSecret(secret, name, value)
      .then(() => {
        toastSuccess(`${name} has been updated`, 'Secret updated');
        navigate('..');
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to update Secret'));
      })
      .finally(() => {
        setIsSaving(false);
      });
  };

  return (
    <>
      <span className="text-lg">Edit Secret</span>
      <form className="my-5 grid grid-cols-6 gap-x-5">
        <hr className="col-span-6 mb-8 text-gray-100" />
        <label className="col-span-2 mt-2 text-gray-700" htmlFor="secret-name">
          Key
        </label>
        <div className="col-span-4 flex flex-col">
          <input
            {...register('name', {
              required: { value: true, message: 'Key is required' },
              pattern: {
                value: /^[a-zA-Z0-9._-]{1,100}$/i,
                message: 'Name must only contain alphanumeric, dash or underscore characters'
              }
            })}
            aria-invalid={errors.name ? 'true' : 'false'}
            className="solid-input"
            id="secret-name"
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
        <hr className="col-span-6 my-8 text-gray-100" />
        <label className="col-span-2 mt-2 text-gray-700" htmlFor="secret-name">
          Value
        </label>
        <div className="col-span-4 flex flex-col">
          <input
            {...register('value', {
              required: { value: true, message: 'Value is required' }
            })}
            aria-invalid={errors.value ? 'true' : 'false'}
            className="solid-input"
            onChange={valueChanged}
            placeholder="●●●●●●●●"
            type="text"
            value={value}
          />
          {errors.value && (
            <p className="my-1 text-sm text-red-600" role="alert">
              {errors.value?.message}
            </p>
          )}
        </div>
        <hr className="col-span-6 mt-8 text-gray-100" />
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
