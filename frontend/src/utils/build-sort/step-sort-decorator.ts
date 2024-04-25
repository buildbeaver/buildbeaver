import { IStep } from '../../interfaces/step.interface';
import { SortDecorator } from './sort-decorator';

/**
 * Provides extra behaviour to steps when sorting a build.
 */
export class StepSortDecorator extends SortDecorator<IStep> {
  get dependencyKeys(): string[] {
    return this.node.depends?.map((dependency) => dependency.step_name) ?? [];
  }

  get name(): string {
    return this.node.name;
  }
}
