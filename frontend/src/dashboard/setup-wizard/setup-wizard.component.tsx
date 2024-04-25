import React, { useContext, useState } from 'react';
import { useSetupStatus } from '../../hooks/setup-status/setup-status.hook';
import { Button } from '../../components/button/button.component';
import { IoChevronForwardSharp, IoWarning } from 'react-icons/io5';
import { FaCheckCircle } from 'react-icons/fa';
import { SimpleContentLoader } from '../../components/content-loaders/simple/simple-content-loader';
import { NavLink } from 'react-router-dom';
import { StructuredError } from '../../components/structured-error/structured-error.component';
import { SetUpRepos } from './set-up-repos.component';
import { SetUpRunners } from './set-up-runners.component';
import { SetUpBuildBeaver } from './set-up-buildbeaver.component';
import { SetUpBuilds } from './set-up-builds.component';
import { SetupContext } from '../../contexts/setup/setup.context';
import { SetupComplete } from './setup-complete.component';

enum Step {
  BuildBeaverInstalled = 'buildbeaver_installed',
  ReposEnabled = 'repos_enabled',
  RunnersRegistered = 'runners_registered',
  BuildsRun = 'builds_run'
}

const steps = [Step.BuildBeaverInstalled, Step.ReposEnabled, Step.RunnersRegistered, Step.BuildsRun];
const labels = {
  [Step.BuildBeaverInstalled]: 'Install the GitHub app',
  [Step.ReposEnabled]: 'Enable repos',
  [Step.RunnersRegistered]: 'Register runners',
  [Step.BuildsRun]: 'Start building'
};

export function SetupWizard(): JSX.Element {
  const { setupUrl } = useContext(SetupContext);
  const [isFirstTimeLoad, setIsFirstTimeLoad] = useState(true);
  const [currentStep, setCurrentStep] = useState<Step | undefined>();
  const { setupStatus, setupStatusError, setupStatusLoading, setupStatusRefreshing, refreshSetupStatus } = useSetupStatus(
    setupUrl,
    currentStep === Step.BuildBeaverInstalled || currentStep === Step.BuildsRun
  );

  const stepNumber = currentStep && steps.indexOf(currentStep) + 1;
  const isCurrentStepComplete = setupStatus && currentStep && setupStatus[currentStep];

  if (isFirstTimeLoad && setupStatus) {
    setCurrentStep(steps.find((step) => !setupStatus[step]));
    setIsFirstTimeLoad(false);
  }

  if (!setupStatus || isFirstTimeLoad) {
    return (
      <>
        <SimpleContentLoader numberOfRows={1} rowHeight={50} />
        <br />
        <SimpleContentLoader numberOfRows={1} rowHeight={300} />
        <br />
        <SimpleContentLoader numberOfRows={1} rowHeight={50} />
      </>
    );
  }

  if (setupStatusError) {
    return <StructuredError error={setupStatusError} fallback="Failed to load setup status" />;
  }

  const stepIndicators = (): JSX.Element => {
    return (
      <div className="flex items-center gap-x-2">
        {steps.map((step, index) => {
          return (
            <React.Fragment key={step}>
              <div
                className={`flex h-6 w-6 items-center justify-center rounded-full border-2 border-flushOrange text-sm font-bold ${
                  setupStatus && setupStatus[step] && 'border-mountainMeadow'
                }`}
              >
                {index + 1}
              </div>
              {index < steps.length - 1 && (
                <div>
                  <IoChevronForwardSharp />
                </div>
              )}
            </React.Fragment>
          );
        })}
      </div>
    );
  };

  const isSetupComplete = !currentStep;

  if (isSetupComplete) {
    return (
      <>
        <div className="flex justify-between rounded-md text-gray-600">
          <div className="text-xl">Setup complete</div>
          {stepIndicators()}
        </div>
        <hr className="mt-2" />
        <div className="my-5">
          <SetupComplete />
        </div>
        <hr className="my-2" />
        <div className="flex justify-end">
          <NavLink to="/">
            <Button label="Finish" />
          </NavLink>
        </div>
      </>
    );
  }

  const nextClicked = (): void => {
    setIsFirstTimeLoad(true);
  };

  const stepStatus = (): JSX.Element => {
    return (
      <div
        className={`flex items-center justify-center rounded-full px-3 ${
          isCurrentStepComplete ? 'bg-mountainMeadowTransparent' : 'bg-flushOrangeTransparent'
        }`}
      >
        {isCurrentStepComplete ? (
          <div className="flex items-center gap-x-1 text-mountainMeadow">
            Complete
            <div>
              <FaCheckCircle />
            </div>
          </div>
        ) : (
          <div className="flex items-center gap-x-1 text-flushOrange">
            Action required
            <div>
              <IoWarning size={20} />
            </div>
          </div>
        )}
      </div>
    );
  };

  return (
    <>
      {setupStatusLoading ? (
        <SimpleContentLoader numberOfRows={1} rowHeight={30} />
      ) : (
        <div className="flex justify-between rounded-md text-gray-600">
          <div className="flex items-center gap-x-2">
            <div className="flex h-6 w-6 items-center justify-center rounded-full border-2 border-gray-600">{stepNumber}</div>
            <div className="text-xl">{labels[currentStep]}</div>
            {setupStatusRefreshing ? (
              <div className="w-36">
                <SimpleContentLoader numberOfRows={1} />
              </div>
            ) : (
              stepStatus()
            )}
          </div>
          {stepIndicators()}
        </div>
      )}
      <hr className="mt-2" />
      <div className="my-5">
        {setupStatusLoading ? (
          <SimpleContentLoader numberOfRows={1} rowHeight={300} />
        ) : (
          <>
            {currentStep === Step.BuildBeaverInstalled && <SetUpBuildBeaver />}
            {currentStep === Step.ReposEnabled && <SetUpRepos refreshSetupStatus={refreshSetupStatus} />}
            {currentStep === Step.RunnersRegistered && <SetUpRunners refreshSetupStatus={refreshSetupStatus} />}
            {currentStep === Step.BuildsRun && <SetUpBuilds />}
          </>
        )}
      </div>
      <hr className="my-2" />
      <div className="flex justify-end">
        <Button label="Next" disabled={!isCurrentStepComplete} loading={setupStatusLoading} onClick={nextClicked}></Button>
      </div>
    </>
  );
}
