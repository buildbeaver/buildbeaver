import React, { FormEvent, useContext, useState } from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { Button } from '../../components/button/button.component';
import { useForm } from 'react-hook-form';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { createSecret } from '../../services/secrets.service';
import { useRepoUrl } from '../../hooks/repo-url/repo-url.hook';
import { useRepo } from '../../hooks/repo/repo.hook';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { NotFound } from '../../components/not-found/not-found.component';

/**
 * The editable form values for a Secret
 */
type FormValues = {
  key: string;
  value: string;
};

/**
 * Component used for creating new Secrets for a repo.
 */
export function SecretCreate(): JSX.Element {
  const repoUrl = useRepoUrl();
  const { repo, repoError, repoLoading } = useRepo(repoUrl);
  const [name, setName] = useState('');
  const [value, setValue] = useState('');
  const [isCreating, setIsCreating] = useState(false);

  const { toastError, toastSuccess } = useContext(ToasterContext);

  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<FormValues>({ mode: 'onBlur' });

  if (repoError) {
    return <StructuredError error={repoError} handleNotFound={true} />;
  }

  if (repoLoading) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={200} />;
  }

  if (!repo) {
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

  // Submit the Secret registration to our API
  const onSubmit = async (): Promise<void> => {
    setIsCreating(true);

    await createSecret(repo!.secrets_url, name, value)
      .then(() => {
        toastSuccess(`${name} has been created`, 'Secret created');
        navigate('..');
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to create Secret'));
      })
      .finally(() => {
        setIsCreating(false);
      });
  };

  return (
    <>
      <span className="text-lg">Create a new Secret</span>
      <form className="my-5">
        <div className="grid grid-cols-6 gap-x-5">
          <hr className="col-span-6 mb-8 text-gray-100" />
          <label className="col-span-2 mt-1 text-gray-700" htmlFor="secret-key">
            Secret key
          </label>
          <div className="col-span-4 flex flex-col">
            <input
              {...register('key', {
                required: { value: true, message: 'Key is required' },
                pattern: {
                  value: /^[a-zA-Z0-9._-]{1,100}$/i,
                  message: 'Name must only contain alphanumeric, dash or underscore characters'
                }
              })}
              aria-invalid={errors.key ? 'true' : 'false'}
              className="solid-input"
              id="secret-key"
              onChange={nameChanged}
              type="text"
              value={name}
            />
            {errors.key && (
              <p className="my-1 text-sm text-red-600" role="alert">
                {errors.key?.message}
              </p>
            )}
          </div>
          <hr className="col-span-6 my-8 text-gray-100" />
          <label className="col-span-2 mt-1 text-gray-700" htmlFor="secret-value">
            Secret value
          </label>
          <div className="col-span-4 flex flex-col">
            <input
              {...register('value', {
                required: { value: true, message: 'Value is required' }
              })}
              aria-invalid={errors.value ? 'true' : 'false'}
              className="solid-input"
              id="secret-value"
              onChange={valueChanged}
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
        </div>

        <div className={'mt-6 flex flex-wrap gap-6'}></div>
      </form>
      <div className="flex flex-wrap justify-end gap-6">
        <NavLink to={'..'}>
          <Button label="Cancel" type={'secondary'} />
        </NavLink>
        <Button label="Create" loading={isCreating} onClick={handleSubmit(onSubmit)} />
      </div>
    </>
  );
}
