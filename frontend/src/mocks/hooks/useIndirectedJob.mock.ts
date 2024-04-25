import * as indirectedJobHook from '../../hooks/indirected-job-graph/indirected-job.hook';

export function mockUseIndirectedJob(): void {
  jest.spyOn(indirectedJobHook, 'useIndirectedJobGraph').mockImplementation(() => {
    return {};
  });
}
