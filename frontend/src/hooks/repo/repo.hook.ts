import { useEffect, useState } from 'react';
import { fetchRepo } from '../../services/repos.service';
import { IRepo } from '../../interfaces/repo.interface';
import { IStructuredError } from '../../interfaces/structured-error.interface';

interface IUseRepo {
  repo?: IRepo;
  repoError?: IStructuredError;
  repoLoading: boolean;
}

export function useRepo(url: string): IUseRepo {
  const [repo, setRepo] = useState<IRepo | undefined>(undefined);
  const [repoError, setRepoError] = useState<IStructuredError | undefined>();
  const [repoLoading, setRepoLoading] = useState(true);

  useEffect(() => {
    const runFetchRepo = async (): Promise<void> => {
      setRepoLoading(true);

      await fetchRepo(url)
        .then((response) => {
          setRepo(response);
        })
        .catch((error: IStructuredError) => {
          setRepoError(error);
        })
        .finally(() => {
          setRepoLoading(false);
        });
    };

    runFetchRepo();
  }, []);

  return {
    repo,
    repoError,
    repoLoading
  };
}
