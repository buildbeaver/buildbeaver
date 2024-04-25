import React, { FormEvent, useContext, useState } from 'react';
import { NavLink, useNavigate } from 'react-router-dom';
import { createRunner } from '../../services/runners.service';
import { Button } from '../../components/button/button.component';
import { useForm } from 'react-hook-form';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { CurrentLegalEntityContext } from '../../contexts/current-legal-entity/current-legal-entity.context';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { SetupContext } from '../../contexts/setup/setup.context';
import { Information } from '../../components/information/information.component';

/**
 * The form values for registering a Runner
 */
type FormValues = {
  name: string;
  certificate: string;
};

interface Props {
  cancelClicked?: () => void;
  runnerRegistered?: () => void;
}

/**
 * Component used for registering new Runners.
 */
export function RunnerRegister(props: Props): JSX.Element {
  const { cancelClicked, runnerRegistered } = props;
  const [name, setName] = useState('');
  const [certificate, setCertificate] = useState('');
  const [isLoading, setIsLoading] = useState(false);

  const { currentLegalEntity } = useContext(CurrentLegalEntityContext);
  const { toastError, toastSuccess } = useContext(ToasterContext);
  const { isInSetupContext } = useContext(SetupContext);

  const navigate = useNavigate();
  const {
    register,
    handleSubmit,
    formState: { errors }
  } = useForm<FormValues>({ mode: 'onBlur' });

  // Handle the name input changing
  const nameChanged = (event: FormEvent<HTMLInputElement>): void => {
    const newName = event.currentTarget.value;

    setName(newName);
  };

  // Handle the certificate textarea changing
  const certificateChanged = (event: FormEvent<HTMLTextAreaElement>): void => {
    const newCertificate = event.currentTarget.value;

    setCertificate(newCertificate);
  };

  // Submit the Runner registration to our API
  const onSubmit = async (): Promise<void> => {
    setIsLoading(true);
    const runnerSearchUrl = currentLegalEntity.runner_search_url;
    const runnerAddUrl = runnerSearchUrl.replace('/search', '/');

    await createRunner(runnerAddUrl, { name, client_certificate_pem: certificate })
      .then(() => {
        toastSuccess(`${name} has been created`, 'Runner created');

        if (isInSetupContext && runnerRegistered) {
          runnerRegistered();
        } else {
          navigate('..');
        }
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to create Runner'));
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <>
      <span className="text-lg">Register a new Runner</span>
      <form className="my-5">
        <div className="grid grid-cols-6 gap-x-5">
          <hr className="col-span-6 mb-8 text-gray-100" />
          <label className="col-span-2 mt-1 text-gray-700" htmlFor="runner-name">
            Runner name
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
              placeholder="my-runner"
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
          <label className="col-span-2 mt-1 text-gray-700" htmlFor="runner-certificate">
            Runner client PEM certificate including header and footer
          </label>
          <div className="col-span-4 flex flex-col">
            <textarea
              {...register('certificate', {
                required: { value: true, message: 'Certificate is required' }
              })}
              aria-invalid={errors.certificate ? 'true' : 'false'}
              className="solid-input font-mono"
              id="runner-certificate"
              onChange={certificateChanged}
              placeholder={'-----BEGIN CERTIFICATE-----\n(Your Runner certificate)\n-----END CERTIFICATE-----'}
              rows={10}
              value={certificate}
            ></textarea>
            {errors.certificate && (
              <p className="my-1 text-sm text-red-600" role="alert">
                {errors.certificate?.message}
              </p>
            )}
          </div>
          <div className="col-span-6 mt-5">
            <Information
              text={`To obtain the runner's client certificate please run the 'bb-runner' command and copy-and-paste the certificate from its output.`}
            />
          </div>
          <hr className="col-span-6 mt-8 text-gray-100" />
        </div>
      </form>
      <div className="flex flex-wrap justify-end gap-6">
        {isInSetupContext && cancelClicked ? (
          <Button label="Cancel" type="secondary" onClick={cancelClicked} />
        ) : (
          <NavLink to={'..'}>
            <Button label="Cancel" type="secondary" />
          </NavLink>
        )}
        <Button label="Register" loading={isLoading} onClick={handleSubmit(onSubmit)} />
      </div>
    </>
  );
}
