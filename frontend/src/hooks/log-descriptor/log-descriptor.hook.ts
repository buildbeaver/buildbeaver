import { useEffect, useState } from 'react';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ILogDescriptor } from '../../interfaces/log-descriptor.interface';
import { fetchLogDescriptor } from '../../services/logs.service';

export interface IUseLogDescriptor {
  logDescriptor?: ILogDescriptor;
  logDescriptorError?: IStructuredError;
  logDescriptorLoading: boolean;
}

export function useLogDescriptor(url: string): IUseLogDescriptor {
  const [logDescriptor, setLogDescriptor] = useState<ILogDescriptor | undefined>();
  const [logDescriptorError, setLogDescriptorError] = useState<IStructuredError | undefined>();
  const [logDescriptorLoading, setLogDescriptorLoading] = useState(true);

  useEffect(() => {
    const runFetchLogDescriptor = async (): Promise<void> => {
      setLogDescriptorLoading(true);

      await fetchLogDescriptor(url)
        .then((response) => {
          setLogDescriptor(response);
        })
        .catch((error: IStructuredError) => {
          setLogDescriptorError(error);
        })
        .finally(() => {
          setLogDescriptorLoading(false);
        });
    };

    runFetchLogDescriptor();
  }, [url]);

  return { logDescriptor, logDescriptorError, logDescriptorLoading };
}
