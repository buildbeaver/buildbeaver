import { useEffect, useState } from 'react';
import { ISecret } from '../../interfaces/secret.interface';
import { fetchSecret } from '../../services/secrets.service';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface IUseSecret {
  secret?: ISecret;
  secretError?: IStructuredError;
  secretLoading: boolean;
}

export function useSecret(url: string): IUseSecret {
  const [secret, setSecret] = useState<ISecret | undefined>();
  const [secretError, setSecretError] = useState<IStructuredError | undefined>();
  const [secretLoading, setSecretLoading] = useState(true);

  useEffect(() => {
    const runFetchSecret = async (): Promise<void> => {
      setSecretLoading(true);

      await fetchSecret(url)
        .then((response) => {
          setSecret(response);
        })
        .catch((error: IStructuredError) => {
          setSecretError(error);
        })
        .finally(() => {
          setSecretLoading(false);
        });
    };

    runFetchSecret();
  }, [url]);

  return { secret, secretError, secretLoading };
}
