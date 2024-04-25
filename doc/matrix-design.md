# Matrix Builds - Design

## Introduction

Some build scenarios require a semantically equivalent series of steps to be run with varied combinations of inputs.
This is typically used to test or build software against multiple versions of a runtime/framework, and/or for multiple
platforms/architectures. This problem can be solved by providing a convenient way for the developer to express the combination of inputs,
and having the build system automatically run the steps with all unique input combinations.

Consider the following inputs:
```
architecture: arm64, x86-32, x86-64
golang_version: 1.15, 1.16, 1.17
```

The build system would generate the following matrix:

```
architecture  golang_version
arm64         1.15
arm64         1.16
arm64         1.17
x86-32        1.15
x86-32        1.16
x86-32        1.17
x86-64        1.15
x86-64        1.16
x86-64        1.17
```
The build would be executed 9 times, with a unique combination of inputs each time.

In addition, many systems provide options for explicitly including or excluding specific combinations of inputs.


## BuildBeaver Proposed Design for Matrix Builds

### Terminology

The repetition defined by Matrix properties applies at the Stage level in BuildBeaver. A BuildBeaver configuration file
currently provides *stage definitions* as the top-level elements in the file.

In order to implement matrix builds, top-level elements in the configuration file will either define a *stage*,
resulting in a single stage being queued for execution, or a *matrix of stages*, resulting in many stages being
queued for execution.

## Config Syntax

A new optional `matrix` property is added to the top-level configuration file element, that enables a matrix to
be defined. Each element under the `matrix` property defines a *matrix variable*. Matrix variables can be
referenced within other properties anywhere in the configuration file via templating. By default a new stage will be
queued for each combination of matrix variable values, forming the *matrix of stages* defined by the top level
configuration element. Each stage in a matrix uses the same step definitions, from the configuration file, but
with matrix variables substituted via templating.

Each top-level element has its own independent `matrix` property and will not inherit matrix variables from
other top-level elements in the configuration file. Matrix variables from another stage can be referenced via
dependencies and artifacts, but including these references will not cause any kind of extra builds of the
referencing stage.

```yaml
- name: test
  description: Run all tests
  type: docker
  matrix:
    image: [golang:1.15, golang:1.16, golang:1.17]
    target-os: [windows, linux, mac]
  steps:
    - name: go
      image: ${{ matrix.image }}
      commands:
        - cd backend
        - go test -mod=vendor ./...
      depends:
        - jobs.generate.artifacts
```

### Include and Exclude

Similarly to concepts in [GitHub](https://docs.github.com/en/actions/reference/workflow-syntax-for-github-actions#jobsjob_idstrategymatrix)
actions and [TravisCI](https://docs.travis-ci.com/user/customizing-the-build/#build-matrix) we will support special `include`
and `exclude` sub-properties inside the `matrix` object. Include will add new or override existing permutations of the
matrix, and exclude will remove matching permutations from the matrix. New matrix variables can be introduced as part
of include statements. (It does not make sense to introduce new matrix variables in an exclude statement since they
would be, by definition, excluded from builds.)

### Stage Dependency References

Dependencies can be specified using the `depends` property, and can depend on the completion of specific subsets
of a matrix of stages using the *matrix dependency* syntax. By default, a dependency on a matrix will prevent the
dependent stage from running until all stage runs have completed for combinations of matrix variables for the
referenced matrix.

If a matrix dependency is specified then the dependent stage can be run after only a subset of the stages
for the referenced matrix have completed.

A regular stage dependency remains unchanged, providing backwards compatibility with existing configuration
files, and is of the form:

`<stage-name>`

where *stage-name* is the name of the stage being referenced, and corresponds to the 'name' property of the top level
element in the configuration file that defines the referenced stage.

A matrix stage dependency is of the form:

`<matrix-name>(<variable1>=<value1>,<variable2>=<value2>...)`

where *matrix-name* is the name of the matrix of stages being referenced, and still corresponds to the 'name' 
property of the top level element in the configuration file that defines the referenced matrix of stages.

The *variable* and *value* elements specify value(s) for zero or more variables from the referenced matrix,
known as the *pinned variables*. The dependent stage depends on all stages in the referenced matrix with the specified
value(s) for the pinned variable(s), i.e. all combinations of non-pinned variables. The dependent stage will not be
run until the stages for all these combinations have completed.

If the dependent stage itself is part of a matrix, any matrix variables whose names match a matrix variable in
the referenced matrix will also pin those variables automatically. In the example below, the 'test-windows' stage has
a matrix variable called 'image', so 'test-windows' matrix stages will only depend on 'build' stages where the
value of 'image' is the same (in this case, just one build because the other variable 'target-os' has been
explicitly pinned with the value 'windows').

Example:

```yaml
- name: build
  matrix:
    image: [golang:1.15, golang:1.16, golang:1.17]
    target-os: [windows, linux, mac]

- name: test-go-1.15-mac
  depends: build(image=golang:1.15,target-os=mac)

- name: test-windows
  depends: build(target-os=windows)
  matrix:
    image: [golang:1.15, golang:1.16]

- name: deploy-mac
  depends: build(target-os=mac)

- name: all-done
  depends: build()
```

This results in the following stage runs and dependencies:

```
Stage name        Matrix name   Matrix variable values                 Depends on
----------------------------------------------------------------------------------------------
build.1           build         image=golang:1.15, target-os=windows   (no dependencies)
build.2           build         image=golang:1.15, target-os=linux     (no dependencies)
build.3           build         image=golang:1.15, target-os=mac       (no dependencies)
build.4           build         image=golang:1.16, target-os=windows   (no dependencies)
build.5           build         image=golang:1.16, target-os=linux     (no dependencies)
build.6           build         image=golang:1.16, target-os=mac       (no dependencies)
build.7           build         image=golang:1.17, target-os=windows   (no dependencies)
build.8           build         image=golang:1.17, target-os=linux     (no dependencies)
build.9           build         image=golang:1.17, target-os=mac       (no dependencies)

test-go-1.15-mac  (no matrix)                                          build 3 (image=golang:1.15,target-os=mac)

test-windows.1    test-windows  image=golang:1.15                      build 1 (image=golang:1.15,target-os=windows)
test-windows.2    test-windows  image=golang:1.16                      build 4 (image=golang:1.16,target-os=windows)

deploy-mac        (no matrix)                                          build 3, build 6 and build 9

all-done          (no matrix)                                          all builds from build 1 to build 9
````


### Step Dependency References

Dependencies between steps in the same stage would not be affected by adding matrix builds, since they
would always refer to another step in the same stage as the dependent step.

It is not currently possible to specify non-artifact (i.e. order) dependencies on a specific step in another
stage, although if this was supported we could use a similar syntax to stage dependencies to specify
matrix dependencies:

`stages.<matrix-name>(<variable1>=<value1>,<variable2>=<value2>...).steps.<step-name>`

### Artifact Dependency References

Step dependencies on artifacts from another stage would need to be enhanced to support matrix dependencies.
For artifact dependencies the syntax is already more complex than stage dependencies, but can be extended
in the same way to reference matrix variable values.

Artifact dependencies for a matrix build do not need to be constrained down to a single stage of the
referenced matrix. If not all variables for the referenced matrix are pinned then the step will depend on
multiple other stages, and will be given artifacts from all the stages it depends on.

As with stage dependencies, if the dependent step is part of a matrix then any matrix variables whose
name matches a variable in the referenced matrix will automatically pin that variable to a value. This
reduces the need to explicitly provide values for matrix variables in the `depends` element. The same result
could be achieved by using template variables based on the dependent stage's matrix variables, but this would
introduce extra 'boiler plate' into the configuration file. If the config author doesn't want this matching to
occur they must use different matrix variable names for the two matrices to avoid them being matched.

The following syntax can specify artifact dependencies referencing a matrix of stages:

```
depends:
  - stages.<matrix-name>(<variable1>=<value1>,<variable2>=<value2>...).steps.<step-name>.artifacts.<artifact-name>
  - stages.<matrix-name>(<variable1>=<value1>,<variable2>=<value2>...).artifacts
```

Any number of matrix variable/value pairs can be provided.

These changes are backwards-compatible with existing configuration files since the format for non-matrix artifact
dependencies remains unchanged.

Example:

```yaml
- name: build
  matrix:
    image: [golang:1.15, golang:1.16, golang:1.17]
    target-os: [windows, linux, mac]
  steps:
    - name: go
      image: ${{ matrix.image }}
      commands:
        - go build ./...

- name: test-windows
  depends: build(target-os=windows)
  matrix:
    image: [golang:1.15, golang:1.16]
  steps:
    - name: go
      image: ${{ matrix.image }}
      commands:
        - go test ./...
      depends:
        - jobs.build(target-os=windows).steps.go.artifacts
```

## Database and Model Changes

Database, model and DAG changes required to support matrix builds are minimal. The existing concept of a 'stage'
is closely aligned to the execution of matrix stage for a particular combination of matrix variables.
The matrix definition in the configuration file can be expanded out into a set of stages as
the build is queued, including all required artifact and stage dependencies, and any template variable substitution
required in the configuration file can be performed at the time the build is queued.

Note that this does preclude us from using secrets as matrix variable values, but does allow us to perform
template substitution into almost any field in the configuration file, including commands.

Each stage for a matrix build should be recorded in the database and the DAG as a separate independent 'stage'
in the queue, so that they can be run on different runners. Each stage requires a unique *stage name*.
This can be achieved by numbering the stages from 1 upwards as the matrix definition in the config
file is expanded. The stage names will be of the form:

`<matrix-name>.<stage-number>`

In addition, we can record the matrix variables used for each stage run in the database. This will allow us to
perform queries for stages and steps based on matrix variables, providing flexibility for APIs and UI to show
matrix build progress in different ways. Explicitly recording matrix variables in the database will provide
more scope for dynamically determining the set of stage runs to queue, if we choose to do this in the future
(see the 'Dynamic Builds' section below).

### Database Changes

The only required database schema change is to add a 'matrix_variables' table, providing the matrix name
as well as the names and values of matrix variables for every stage that is part of a matrix.
This table is not required for the build to occur, but can be used for querying for progress and reporting,
and can be used by the UI for showing the build graph.
```
CREATE TABLE IF NOT EXISTS matrix_variables
(
    matrix_variable_id text NOT NULL PRIMARY KEY,
    matrix_variable_stage_id text NOT NULL REFERENCES stages (stages_id)
    matrix_variable_matrix_name text NOT NULL
    matrix_variable_name text NOT NULL
    matrix_variable_value text NOT NULL
)
```

No changes are required to the schema for the stages table, although the stage_name field will now contain
names that look slightly different for stages that are part of a matrix.

Any references to a stage via stage_id remain the same, and will now refer to a specific stage within a matrix.
This means no database changes are required for dependencies. In particular, the 'stages_depend_on_stages' table
can remain unchanged.

Because artifact and stage dependencies are stored in their tables as serialized JSON documents,
no database schema changes are required to support the new dependency syntax.

### Model and DAG Changes

We currently have Directed Acyclic Graphs (DAGs) representing stage and step dependencies. The DAGs use stage names
and step names to refer to other nodes in the graph.

Because each stage within a matrix build remains a fully-specified independent stage, no changes are required in
the DAG for dependency management. Because the names for stages include the stage number as well as the matrix name,
we still have a unique identifier for each stage in the DAG.

Code changes required:

**queue.BuildQueueParser:**

- Change parseStage() to expand out matrix variables into a set of stage objects, one for each stage we wish
to run for this matrix, and number the resulting stages from 1 upwards.

- Change parseStageDependencies() to resolve the matrix build variables specified in dependencies in the
configuration file down to specific stage numbers. This will need to be done after the referenced matrix has
been expanded into a numbered set of stage objects.

- Change parseStepDependencies() to resolve matrix build variables specified in artifact dependencies to
specific stage run numbers. This will need to be done after the referenced matrix has been expanded into a
numbered set of stage objects.

**Additional Changes**

It may be useful to automatically look up matrix variables when returning a stage name via the API, in order to
be able to render something more meaningful than just a matrix name and run number. The matrix_variables table
can be used to look up more detail for a specific stage, especially matrix variable names and values.

## Dynamic vs Static Matrix Builds

This design assumes 'static' matrix builds, which means the set of stages that are required is determined
at the time the build is queued, based entirely on the contents of the configuration file. In this sense it is
a static set of stages, equivalent to fully spelling out the stages explicitly in a (very long) configuration file.

It would also be possible to allow user code to dynamically determine the set of stages for a matrix, providing much
more flexibility in which combinations are actually built. This would involve a stage running early in the build
which would queue more stages. In this case it would be very helpful to track matrix variables explicitly in the
database, and make them visible to this early stage. The stage could also define new matrix variables via a
suitable API, and these could be recorded in the database so the BuildBeaver UI would have visibility to show progress.

# Questions and Answers

The following questions are from the original matrix design notes, together with answers based on the design above.

> Are stages in the matrix distributed across runners?

Different stage runs for a matrix are each represented as a fully-formed stage, and so can be distributed across
different runners. Each step within a given stage run will still be run on the same runner.

> Do dependent stages/steps run after all permutations or a nominated subset?

By default dependent stages/steps run after all permutations of the referenced matrix of stages have completed,
but a subset can be explicitly nominated via *matrix dependencies* (see above).

> Do dependent stages/steps also automatically form a matrix of upstream inputs?

Dependent stages/steps do not automatically form their own matrix if an upstream input (i.e. dependency)
is a matrix. Any matrix for a dependent stage must be explicitly defined by the developer in the configuration
file.

If a matrix is specified for a dependent stage or step then the declared dependency can use templating with
the matrix variables inside the `depends` element to constrain the dependency to a subset of the stage runs
for the referenced stage.

> How do downstream stages/steps nominate the specific upstream artifacts they depend on?

Artifact dependencies can refer to a specific upstream artifact by resolving the dependency down to a single stage
in the referenced matrix. This is done by pinning all matrix variables for the referenced matrix to specific values.
Alternatively some matrix variables can be left un-pinned, and artifacts for all matching stages will be provided.
