# Workflows - Design

## Introduction

This document gives a design and theory of operations for the concept of a *workflow* within BuildBeaver. Workflows
are groups of jobs within a build that are partially independent of each other.

The bb command-line tool can be used to run only one or more selected workflows, rather than every workflow in
the build.

## Requirements

Key requirements:

1. To be able to run only a subset of workflows (or only one workflow) from a build via the
bb command-line tool, and potentially from a build server.

1. All required workflows must be executed, even if not explicitly named in the set of workflows to run. For example, 
if requested workflow B requires workflow A to be run in order to function, both workflows A and B should be run.

1. Minimal changes to the existing BB model would be desirable (but not essential)

Things that are NOT a requirement:

4. The ability to wait for one workflow to fully complete before starting another workflow. Such requirements can be
addressed via job dependencies.

1. The ability for one workflow to 'call' another workflow, like a subroutine. This can be achieved through
language-specific features when submitting jobs dynamically, and via regular job dependencies.

## BuildBeaver Proposed Design for Workflows

Workflows are defined implicitly rather than explicitly, and do not need to be defined in the YAML file
in advance of being used.

## Job Definition Syntax

A workflow is defined by prefixing the names of jobs in that workflow with the workflow name, with a dot separator.
This means that job names will have the following format:

`<workflow-name>.<job-name>`

A job without a workflow prefix in its name is considered part of the *default workflow* which is always run.

The same syntax can be used in job dependencies to refer to jobs within another workflow.
NOTE: We may need to tweak the dependency syntax to be able to tell whether a dependency includes a workflow prefix
or not.

For example, the following YAML defines three workflows:
1. `generate` which includes two jobs, named `generate-go-code` and `generate-java-code`
2. `tests` which includes one job named `unit-tests`
3. `build` which is defined in the *depends* clause and is expected to include a job named `build-go-code`

```yaml
- jobs:
  - name: generate.generate-go-code
    description: Re-generates Go code
    type: docker
    steps:
      - name: generate-go
  - name: generate.generate-java-code
    description: Re-generates Java code
    type: docker
    steps:
      - name: generate-go
  - name: tests.unit-tests
    description: Run all unit tests
    type: docker
    depends: [ generate.generate-go-code, build.build-go-code ]
    steps:
      - name: go-tests
```

### Determining Workflows to Run

The build server maintains a dynamically changing list of workflows to run during the build, known as the
*current workflow list*. A job is eligible to be dequeued and run if it is associated with a workflow in the
current workflow list, or if it is not associated with any workflow (and therefore logically part of the default
workflow).

When a build is kicked off via bb (and eventually via a build server) the user can optionally specify a set of
workflows to run, forming the initial value for the current workflow list. If no set of workflows is specified then
all workflows will be run.

There is no explicit mechanism for defining dependencies between workflows; instead, workflow dependencies are
inferred via job dependencies.

All jobs defined in the YAML are parsed and stored in the database, together with the workflow each job
is associated with as defined in the job's name field. If a job is eligible to be run (because it is part of a workflow that
is in the current workflow list) and that job depends on a job in another workflow, then the referenced workflow
is also added to the current workflow list. This ensures that all jobs in the referenced workflow will also be run,
allowing the job dependency to be satisfied.

As jobs are added dynamically they are subject to the same rules as statically-defined jobs.
Any dynamically added job that is associated with a workflow in the current workflow list is eligible to be run, and
if that newly added job depends on a job in another workflow then the referenced workflow is dynamically added to the
current workflow list, making its own jobs eligible to be run.

### Dynamic Jobs and 'Jobinators'

Any job which uses the BuildBeaver dynamic SDK to dynamically create other jobs will be referred to as a *jobinator*.

Jobinators are defined in the same way as ordinary jobs, either statically in the YAML file or dynmaically
by another jobinator. A Jobinator can optionally nominate a workflow in its name, and will only be run if that workflow
is on the current workflow list.

A Jobinator can submit jobs to any workflow, and does not need to be associated with a workflow itself. 
Alternatively it is possible to have one Jobinator per workflow if that's how the customer wishes to structure
their builds.

When a Jobinator is executed, it will be passed the current workflow list which it can examine to determine
which jobs it should dynamically submit. Because the current workflow list can expand over time, the Jobinator
should also subscribe to changes in this list, to be told when new workflows are added. If the Jobinator is
responsible for submitting jobs for the new workflow then it can go ahead and submit more jobs.

As is usual in the Dynamic SDK, the ability to be notified when a new workflow is added to the current workflow list
will be provided as a callback option (on the build), or a wait function to wait for the next workflow to be added,
which would typically need to be used on a separate thread.

### DAG validation and cross-workflow job dependencies

Often jobs in a workflow will be dependent on jobs in another workflow, with all these jobs being
submitted dynamically. Different Jobinators can potentially be used for submitting jobs for each workflow.

Because of the dynamic nature of these scenarios, validation of job dependencies between workflows must be able to
be deferred until after the jobs are parsed. Very often the dependency itself causes the referenced job to
be created at a later time, by adding the referenced job's workflow to the current workflow list, allowing the
referenced workflow's Jobinator to be run and then submit the referenced job dynamically.

When storing a set of jobs in the database, we require jobs referenced in job dependencies within the same workflow
to already exist, either by already being in the database or by being in the same set of jobs being
dynamically created.

If a job references a job in another workflow as a dependency, and that referenced job already exists, then it
can be stored as a dependency in the database in the usual manner (via JobID). However, if the referenced job
does not yet exist, the dependency needs to be stored in text form as a *deferred cross-workflow dependency*.

Deferred cross-workflow dependencies are ignored when validating a DAG, preventing an immediate validation failure when
submitting the dependent job. Jobs with deferred cross-workflow dependencies are not eligible to be dequeued
and run, since the job they depend on in the other workflow has not yet been submitted.

When new jobs are submitted via the CreateJobs() dynamic API function, any deferred cross-workflow dependencies
that refer to a newly submitted job will be *resolved* into a regular job dependency, specified via the JobID of
the new job. The dependent job will then become eligible to be dequeued and run in the normal way, after the
referenced job has completed.

###
The promotion of deferred cross-workflow dependencies into regular job dependencies
allows validation to prevent cycles in the job graph; if a new job would create such a cycle (even indirectly)
then it can be rejected and the CreateJobs() API call can be failed.


## Remaining Questions

1. Can job dependencies do the job of workflow dependencies? I.e. can workflow dependencies be implicit?
[Mark: I think yes; we can possibly add the ability to explicitly declare workflow dependencies later, if there is some
reason to, but this would just be a means to 'pull in' other workflows when one workflow is nominated to be run
rather than giving the ability for one workflow to wait until another is *completed*. ]

1. Can workflow declarations themselves be implicit, obtained from job names and job dependencies without the need
to ever explicitly declare workflows? [Mark: I think probably yes, although it may be useful to be able to annotate workflow
names with descriptions or other attributes]

1. Should we explicitly specify a *workflow* field when defining a job, rather than just using the dot syntax in
it's name? Would this be clearer? e.g:
```yaml
- jobs:
    - name: generate-go-code
      workflow: generate
      description: Re-generates all generated go code
      type: docker
      steps:
        - name: generate-go
    - name: unit-tests
      workflow: tests
      description: Run all unit tests
      type: docker
      depends: [ generate.generate-go-code, build.build-go-code ]
      steps:
        - name: go-tests
```
