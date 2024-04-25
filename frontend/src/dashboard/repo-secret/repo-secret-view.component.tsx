import React, { useContext, useState } from 'react';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { Button } from '../../components/button/button.component';
import { NavLink, useLocation, useNavigate } from 'react-router-dom';
import { Dialog } from '../../components/dialog/dialog.component';
import { IoAlertCircleSharp } from 'react-icons/io5';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { NotFound } from '../../components/not-found/not-found.component';
import { useSecretUrl } from '../../hooks/secret-url/secret-url.hook';
import { useSecret } from '../../hooks/secret/secret.hook';
import { deleteSecret } from '../../services/secrets.service';
import { createdAt } from '../../utils/build.utils';
import { removeLastPathPart } from '../../utils/path.utils';

export function RepoSecretView(): JSX.Element {
  const { toastError, toastSuccess } = useContext(ToasterContext);
  const location = useLocation();
  const navigate = useNavigate();
  const secretUrl = useSecretUrl();
  const { secret, secretError, secretLoading } = useSecret(secretUrl);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  if (secretError) {
    return <StructuredError error={secretError} handleNotFound={true} />;
  }

  if (secretLoading) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={200} />;
  }

  if (!secret) {
    return <NotFound />;
  }

  // Delete Secret handler -> open the delete dialog
  const deleteClicked = () => {
    setDeleteDialogOpen(true);
  };

  // Delete dialog delete clicked handler -> Call API to delete
  const deleteSecretClicked = async () => {
    await deleteSecret(secret)
      .then(() => {
        toastSuccess(`${secret.name} has been deleted`, 'Secret deleted');
        navigate(removeLastPathPart(location.pathname));
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to delete Secret'));
      });
  };

  const rows: { content: JSX.Element; label: string }[] = [
    { label: 'Key', content: <span>{secret.name}</span> },
    { label: 'Created', content: <span>{createdAt(secret.created_at)}</span> },
    { label: 'Updated', content: <span>{createdAt(secret.updated_at)}</span> }
  ];

  return (
    <>
      <div className="flex items-center justify-between">
        <span className="text-lg">Secret details</span>
        <div className="flex gap-x-5">
          <NavLink to="edit">
            <Button label="Edit" size="small" />
          </NavLink>
          <Button label="Delete" size="small" type="danger" onClick={() => deleteClicked()} />
        </div>
      </div>
      <div className="my-5 grid grid-cols-6">
        {rows.map((row, index) => {
          const style = index % 2 === 0 ? 'border-y bg-gray-50 border-gray-100' : '';
          return (
            <React.Fragment key={row.label}>
              <div className={`col-span-2 py-5 px-2 ${style}`}>{row.label}</div>
              <div className={`col-span-4 py-5 px-2 ${style}`}>{row.content}</div>
            </React.Fragment>
          );
        })}
      </div>
      <Dialog
        actionButton={{
          text: 'Delete',
          type: 'danger',
          clicked: () => deleteSecretClicked()
        }}
        icon={<IoAlertCircleSharp className="text-amaranth" size={24} />}
        isOpen={deleteDialogOpen}
        setIsOpen={setDeleteDialogOpen}
        title={'Delete Secret'}
      >
        <p className="text-sm text-gray-500">Are you sure you want to secret {secret.name}?</p>
      </Dialog>
    </>
  );
}
