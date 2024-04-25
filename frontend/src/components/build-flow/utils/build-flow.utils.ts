import { BuildFlowConstants } from '../constants/build-flow.constants';

export class BuildFlowUtils {
  static calculateStepYPosition(modifier: number): number {
    return (
      modifier * (BuildFlowConstants.STEP.HEIGHT + BuildFlowConstants.STEP.MARGIN) +
      BuildFlowConstants.STEP.MARGIN * 2 +
      BuildFlowConstants.JOB.HEIGHT
    );
  }
}
