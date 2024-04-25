import { IStep } from '../../interfaces/step.interface';
import { StepSortDecorator } from './step-sort-decorator';

describe('step-sort-decorator', () => {
  it('should extract dependency keys from a step', () => {
    const step = {
      name: 'python-builder',
      depends: [
        {
          step_name: 'go-builder'
        }
      ]
    } as IStep;

    const stepSortDecorator = new StepSortDecorator(step);

    expect(stepSortDecorator.name).toBe('python-builder');
    expect(stepSortDecorator.dependencyKeys).toEqual(['go-builder']);
  });
});
