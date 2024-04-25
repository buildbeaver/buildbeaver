import React, { useContext, useState } from 'react';
import { useRunner } from '../../hooks/runner/runner.hook';
import { useRunnerUrl } from '../../hooks/runner-url/runner-url.hook';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { PlatformIndicator } from '../platform-indicator/platform-indicator.component';
import { Button } from '../../components/button/button.component';
import { NavLink, useNavigate } from 'react-router-dom';
import { Dialog } from '../../components/dialog/dialog.component';
import { IoAlertCircleSharp } from 'react-icons/io5';
import { deleteRunner } from '../../services/runners.service';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { NotFound } from '../../components/not-found/not-found.component';
import { createdAt } from '../../utils/build.utils';

export function RunnerView(): JSX.Element {
  const { toastError, toastSuccess } = useContext(ToasterContext);
  const navigate = useNavigate();
  const runnerUrl = useRunnerUrl();
  const { runner, runnerError, runnerLoading } = useRunner(runnerUrl);
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);

  if (runnerError) {
    return <StructuredError error={runnerError} handleNotFound={true} />;
  }

  if (runnerLoading) {
    return <SimpleContentLoader numberOfRows={1} rowHeight={200} />;
  }

  if (!runner) {
    return <NotFound />;
  }

  // Delete Runner handler -> open the delete dialog
  const deleteClicked = () => {
    setDeleteDialogOpen(true);
  };

  // Delete dialog delete clicked handler -> Call API to delete
  const deleteRunnerClicked = async () => {
    await deleteRunner(runner)
      .then(() => {
        toastSuccess(`${runner.name} has been deleted`, 'Runner deleted');
        navigate('..');
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to delete Runner'));
      });
  };

  const badges = (items: string[]): JSX.Element => {
    return (
      <div className="col-span-4 flex flex-wrap gap-x-2 gap-y-2">
        {items.map((item) => (
          <span className="rounded-md border bg-blue-100 py-1 px-2 text-sm" key={item}>
            {item}
          </span>
        ))}
      </div>
    );
  };

  const notReportedMessage = 'Runner has not reported yet';
  const rows: { content: JSX.Element; label: string }[] = [
    { label: 'Name', content: <span>{runner.name}</span> },
    {
      label: 'Operating system',
      content: (
        <div className="flex items-center gap-x-1">
          <div>
            <PlatformIndicator runsOn={[runner.operating_system, runner.architecture]} />
          </div>
          <span>{runner.operating_system}</span>
        </div>
      )
    },
    { label: 'Architecture', content: <span>{runner.architecture || notReportedMessage}</span> },
    { label: 'Software version', content: <span className="font-mono">{runner.software_version || notReportedMessage}</span> },
    { label: 'Created', content: <span>{createdAt(runner.created_at)}</span> },
    { label: 'Job types', content: badges(runner.supported_job_types) },
    { label: 'Labels', content: badges(runner.labels) },
    {
      label: 'Enabled',
      content: (
        <div className="flex items-center gap-x-1">
          <input type="checkbox" disabled={true} checked={runner.enabled} />
        </div>
      )
    }
  ];

  return (
    <>
      <div className="flex items-center justify-between">
        <span className="text-lg">Runner details</span>
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
          clicked: () => deleteRunnerClicked()
        }}
        icon={<IoAlertCircleSharp className="text-amaranth" size={24} />}
        isOpen={deleteDialogOpen}
        setIsOpen={setDeleteDialogOpen}
        title={'Delete Runner'}
      >
        <p className="text-sm text-gray-500">Are you sure you want to delete {runner.name}?</p>
      </Dialog>
    </>
  );
}
