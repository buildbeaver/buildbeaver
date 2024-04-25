import React, { useContext } from 'react';
import { Button } from '../../components/button/button.component';
import { IRepo } from '../../interfaces/repo.interface';
import { updateRepo } from '../../services/repos.service';
import { NavLink } from 'react-router-dom';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { UpdateToken } from '../../models/update-token.model';
import { SetupContext } from '../../contexts/setup/setup.context';

interface Props {
  isLoading: boolean;
  repo: IRepo;
  registerUpdateToken: (token: UpdateToken) => void;
  repoUpdated: () => void;
}

export function RepoListItem(props: Props): JSX.Element {
  const { isLoading, repo, registerUpdateToken, repoUpdated } = props;
  const { toastError, toastSuccess } = useContext(ToasterContext);
  const { isInSetupContext } = useContext(SetupContext);

  const setRepoEnabled = async (enabled: boolean): Promise<void> => {
    const data = { enabled };
    const updateToken = new UpdateToken(repo.id);

    registerUpdateToken(updateToken);

    await updateRepo(repo, data)
      .then(() => {
        repoUpdated();
        toastSuccess(`${repo.name} ${enabled ? 'enabled' : 'disabled'}`, 'Repo updated');
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to update repo'));
      })
      .finally(() => {
        updateToken.end();
      });
  };

  const itemContents = (): JSX.Element => {
    let label;
    let clickHandler;

    if (repo.enabled) {
      label = 'Disable';
      clickHandler = () => setRepoEnabled(false);
    } else {
      label = 'Enable';
      clickHandler = () => setRepoEnabled(true);
    }

    return (
      <>
        <div className="flex flex-col">
          <strong className={`${!isInSetupContext && 'cursor-pointer'}`}>{repo.name}</strong>
          <span>
            {repo.description === '' ? <span className="text-gray-300">No description available</span> : repo.description}
          </span>
        </div>
        <Button label={label} loading={isLoading} onClick={clickHandler} size="regular" />
      </>
    );
  };

  const commonStyles = 'flex items-center justify-between p-4 text-sm';

  return isInSetupContext ? (
    <div className={commonStyles}>{itemContents()}</div>
  ) : (
    <NavLink className={commonStyles} to={repo.name}>
      {itemContents()}
    </NavLink>
  );
}
