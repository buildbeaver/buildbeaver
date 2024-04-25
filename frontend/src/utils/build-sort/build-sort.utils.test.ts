import { IBuildGraph } from '../../interfaces/build-graph.interface';
import { sortBuild } from './build-sort.utils';

describe('build-sort.utils', () => {
  describe('sortBuild()', () => {
    it('should sort jobs and steps within a build', () => {
      const bGraph = {
        // A valid build graph must include a build object
        build: {
          id: 'build:aa68fd16-b6c7-4f9e-8390-4154238a50aa'
        },
        jobs: [
          {
            job: {
              name: 'js-test',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'generate',
                  artifact_dependencies: undefined
                }
              ]
            },
            steps: [
              {
                name: 'yarn',
                depends: []
              },
              {
                name: 'npm',

                depends: [
                  {
                    step_name: 'yarn'
                  }
                ]
              },
              {
                name: 'jest',
                depends: []
              },
              {
                name: 'javascript',
                depends: []
              }
            ]
          },
          {
            job: {
              name: 'build',
              depends: [
                {
                  job_name: 'generate',
                  artifact_dependencies: [
                    {
                      job_name: 'generate',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                },
                {
                  job_name: 'js-test',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'go-test-sqlite',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'go-test-postgres',
                  artifact_dependencies: undefined
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'go-test-postgres',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'generate',
                  artifact_dependencies: [
                    {
                      job_name: 'generate',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'do-thing',
              depends: undefined
            },
            steps: [
              {
                name: 'python-builder',
                depends: [
                  {
                    step_name: 'go-builder'
                  }
                ]
              },
              {
                name: 'go-builder',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'generate',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                }
              ]
            },
            steps: [
              {
                name: 'generate',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'go-test-sqlite',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'generate',
                  artifact_dependencies: [
                    {
                      job_name: 'generate',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'base',
              depends: undefined
            },
            steps: [
              {
                name: 'go-builder',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'package',
              depends: [
                {
                  job_name: 'build',
                  artifact_dependencies: [
                    {
                      job_name: 'build',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          }
        ]
      } as IBuildGraph;

      const sortedBuild = sortBuild(bGraph);

      expect(sortedBuild.jobs).toBeDefined();

      const jobs = sortedBuild.jobs!;

      expect(jobs).toHaveLength(8);

      expect(jobs[0].job.name).toBe('do-thing');
      expect(jobs[0].steps).toHaveLength(2);
      expect(jobs[0].steps[0].name).toBe('go-builder');
      expect(jobs[0].steps[1].name).toBe('python-builder');

      expect(jobs[1].job.name).toBe('base');
      expect(jobs[1].steps).toHaveLength(1);
      expect(jobs[1].steps[0].name).toBe('go-builder');

      expect(jobs[2].job.name).toBe('generate');
      expect(jobs[2].steps).toHaveLength(1);
      expect(jobs[2].steps[0].name).toBe('generate');

      expect(jobs[3].job.name).toBe('go-test-postgres');
      expect(jobs[3].steps).toHaveLength(1);
      expect(jobs[3].steps[0].name).toBe('go');

      expect(jobs[4].job.name).toBe('go-test-sqlite');
      expect(jobs[4].steps).toHaveLength(1);
      expect(jobs[4].steps[0].name).toBe('go');

      expect(jobs[5].job.name).toBe('js-test');
      expect(jobs[5].steps).toHaveLength(4);
      expect(jobs[5].steps[0].name).toBe('javascript');
      expect(jobs[5].steps[1].name).toBe('jest');
      expect(jobs[5].steps[2].name).toBe('yarn');
      expect(jobs[5].steps[3].name).toBe('npm');

      expect(jobs[6].job.name).toBe('build');
      expect(jobs[6].steps).toHaveLength(1);
      expect(jobs[6].steps[0].name).toBe('go');

      expect(jobs[7].job.name).toBe('package');
      expect(jobs[7].steps).toHaveLength(1);
      expect(jobs[7].steps[0].name).toBe('go');
    });

    it('should sort jobs and steps within a build V2', () => {
      const bGraph = {
        // A valid build graph must include a build object
        build: {
          id: 'build:bb68fd16-b6c7-4f9e-8390-4154238a50bb'
        },
        jobs: [
          {
            job: {
              name: 'do-thing',
              depends: undefined
            },
            steps: [
              {
                name: 'go-builder',
                depends: undefined
              },
              {
                name: 'python-builder',
                depends: [
                  {
                    step_name: 'go-builder'
                  }
                ]
              }
            ]
          },
          {
            job: {
              name: 'generate',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                }
              ]
            },
            steps: [
              {
                name: 'generate',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'go-test-postgres',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'generate',
                  artifact_dependencies: [
                    {
                      job_name: 'generate',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'go-test-sqlite',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'generate',
                  artifact_dependencies: [
                    {
                      job_name: 'generate',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'base',
              depends: undefined
            },
            steps: [
              {
                name: 'go-builder',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'js-test',
              depends: [
                {
                  job_name: 'base',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'generate',
                  artifact_dependencies: undefined
                }
              ]
            },
            steps: [
              {
                name: 'jest',
                depends: []
              },
              {
                name: 'javascript',
                depends: []
              },
              {
                name: 'yarn',
                depends: []
              },
              {
                name: 'npm',
                depends: [
                  {
                    step_name: 'yarn'
                  }
                ]
              }
            ]
          },
          {
            job: {
              name: 'build',
              depends: [
                {
                  job_name: 'generate',
                  artifact_dependencies: [
                    {
                      job_name: 'generate',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                },
                {
                  job_name: 'js-test',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'go-test-sqlite',
                  artifact_dependencies: undefined
                },
                {
                  job_name: 'go-test-postgres',
                  artifact_dependencies: undefined
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          },
          {
            job: {
              name: 'package',
              depends: [
                {
                  job_name: 'build',
                  artifact_dependencies: [
                    {
                      job_name: 'build',
                      step_name: '',
                      group_name: ''
                    }
                  ]
                }
              ]
            },
            steps: [
              {
                name: 'go',
                depends: undefined
              }
            ]
          }
        ]
      } as IBuildGraph;

      const sortedBuild = sortBuild(bGraph);

      expect(sortedBuild.jobs).toBeDefined();

      const jobs = sortedBuild.jobs!;

      expect(sortedBuild.jobs).toHaveLength(8);

      expect(jobs[0].job.name).toBe('do-thing');
      expect(jobs[0].steps).toHaveLength(2);
      expect(jobs[0].steps[0].name).toBe('go-builder');
      expect(jobs[0].steps[1].name).toBe('python-builder');

      expect(jobs[1].job.name).toBe('base');
      expect(jobs[1].steps).toHaveLength(1);
      expect(jobs[1].steps[0].name).toBe('go-builder');

      expect(jobs[2].job.name).toBe('generate');
      expect(jobs[2].steps).toHaveLength(1);
      expect(jobs[2].steps[0].name).toBe('generate');

      expect(jobs[3].job.name).toBe('js-test');
      expect(jobs[3].steps).toHaveLength(4);
      expect(jobs[3].steps[0].name).toBe('javascript');
      expect(jobs[3].steps[1].name).toBe('jest');
      expect(jobs[3].steps[2].name).toBe('yarn');
      expect(jobs[3].steps[3].name).toBe('npm');

      expect(jobs[4].job.name).toBe('go-test-sqlite');
      expect(jobs[4].steps).toHaveLength(1);
      expect(jobs[4].steps[0].name).toBe('go');

      expect(jobs[5].job.name).toBe('go-test-postgres');
      expect(jobs[5].steps).toHaveLength(1);
      expect(jobs[5].steps[0].name).toBe('go');

      expect(jobs[6].job.name).toBe('build');
      expect(jobs[6].steps).toHaveLength(1);
      expect(jobs[6].steps[0].name).toBe('go');

      expect(jobs[7].job.name).toBe('package');
      expect(jobs[7].steps).toHaveLength(1);
      expect(jobs[7].steps[0].name).toBe('go');
    });

    it('should not throw when sorting a build with no jobs', () => {
      const bGraph = {
        build: {
          id: 'build:aa68fd16-b6c7-4f9e-8390-4154238a50aa'
        }
      } as IBuildGraph;

      expect(() => sortBuild(bGraph)).not.toThrow();
    });
  });
});
