import React, { useContext, useState } from 'react';
import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { BuildStatusIndicator } from '../build-status-indicator/build-status-indicator.component';
import Gravatar from 'react-gravatar';
import { IoCalendarClearOutline, IoRepeat, IoStopwatchOutline } from 'react-icons/io5';
import { backgroundColour, createdAt, trimRef, trimSha } from '../../utils/build.utils';
import { IStructuredError } from '../../interfaces/structured-error.interface';
import { ToasterContext } from '../../contexts/toaster/toaster.context';
import { useNavigate } from 'react-router-dom';
import { Button } from '../../components/button/button.component';
import { createBuild } from '../../services/builds.service';
import { useAnyLegalEntity } from '../../hooks/any-legal-entity/any-legal-entity.hook';
import { isFinished } from '../../utils/build-status.utils';
import { Timer } from '../../components/timer/timer.component';
import { getStructuredErrorMessage } from '../../utils/error.utils';
import { Dialog } from '../../components/dialog/dialog.component';
import { Checkbox } from '../../components/checkbox/checkbox.component';
import { Status } from '../../enums/status.enum';
import { Information } from '../../components/information/information.component';

interface Props {
  buildGraph: IBuildGraph;
}

/**
 * Displays the metadata for a build, including any build level error messages.
 */
export function BuildMetadata(props: Props): JSX.Element {
  const { buildGraph } = props;
  const [isLoading, setIsLoading] = useState(false);
  const [runAgainDialogOpen, setRunAgainDialogOpen] = useState(false);
  const [forceRebuild, setForceRebuild] = useState(buildGraph.build.status === Status.Succeeded);
  const { toastError, toastSuccess } = useContext(ToasterContext);
  const navigate = useNavigate();
  const legalEntity = useAnyLegalEntity();

  const onBuildAgainClicked = async (): Promise<void> => {
    setIsLoading(true);
    await createBuild(buildGraph.repo.builds_url, { from_build_id: buildGraph.build.id, opts: { force: forceRebuild } })
      .then((newBuildGraph) => {
        toastSuccess(`Build #${newBuildGraph.build.name} has been queued`, 'Build Queued');
        navigate(`/${legalEntity.type}/${legalEntity.name}/repos/${newBuildGraph.repo.name}/builds/${newBuildGraph.build.name}`);
      })
      .catch((error: IStructuredError) => {
        toastError(getStructuredErrorMessage(error, 'Failed to enqueue build'));
      })
      .finally(() => {
        setIsLoading(false);
      });
  };

  return (
    <div className="flex whitespace-nowrap">
      <div className={`w-[4px] min-w-[4px] ${backgroundColour(buildGraph.build.status)}`}></div>
      <div className="flex gap-x-2 p-3">
        <div className="flex flex-col items-center gap-y-0.5">
          <BuildStatusIndicator status={buildGraph.build.status} size={20}></BuildStatusIndicator>
          <Gravatar email={buildGraph.commit.author_email} className="h-[20px] w-[20px] rounded-full" />
        </div>
        <div className="flex flex-col gap-y-1 text-sm">
          <div title={`Build #${buildGraph.build.name}`}>
            <a target="_blank" href={buildGraph.commit.link} rel="noopener noreferrer">
              <strong>
                <span title="Git Ref">{trimRef(buildGraph.build.ref)}</span>
              </strong>
              {' - '}
              <span title="Git SHA">{trimSha(buildGraph.commit.sha)}</span>
            </a>
          </div>
          <span className="text-sm">
            Committed by <strong>{buildGraph.commit.author_name}</strong>
          </span>
        </div>
      </div>
      <div className="flex w-[150px] min-w-[150px] flex-col justify-center gap-y-1 p-3 text-sm">
        <div className="flex items-center gap-x-2" title="Created">
          <IoCalendarClearOutline className="ml-[1px]" size={14} />
          {createdAt(buildGraph.build.created_at)}
        </div>
        <div className="flex items-center gap-x-2" title="Duration">
          <IoStopwatchOutline size={16} />
          <Timer timings={buildGraph.build.timings} />
        </div>
      </div>
      <div className="flex min-w-0 grow flex-col gap-y-1 p-3 text-sm">
        <div className="... truncate" title={buildGraph.commit.message}>
          {buildGraph.commit.message}
        </div>
        {buildGraph.build.error && (
          <div className="... truncate text-sm text-red-500">
            <span title={buildGraph.build.error}>
              <strong>{buildGraph.build.error}</strong>
            </span>
          </div>
        )}
      </div>
      {buildGraph.jobs && buildGraph.jobs.length > 0 && (
        <div className="flex flex-col justify-center">
          <Button
            size="regular"
            disabled={!isFinished(buildGraph.build.status)}
            label="Run Again"
            loading={isLoading}
            onClick={() => setRunAgainDialogOpen(true)}
          />
        </div>
      )}
      <Dialog
        actionButton={{ text: 'Run', type: 'primary', clicked: onBuildAgainClicked }}
        icon={<IoRepeat size={24} />}
        isOpen={runAgainDialogOpen}
        setIsOpen={setRunAgainDialogOpen}
        title={`Run build #${buildGraph.build.name} again`}
      >
        <div className="flex flex-col gap-y-3 py-2">
          <Checkbox
            checked={forceRebuild}
            id="force-rebuild"
            label="Force rebuild of all jobs"
            onChange={() => setForceRebuild(!forceRebuild)}
          />
          <Information text="Check this option to rebuild all jobs even if they completed successfully" />
        </div>
      </Dialog>
    </div>
  );
}
