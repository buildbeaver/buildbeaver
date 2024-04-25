package integration_tests

import (
	"fmt"
	"math/rand"
	"sync"
	"testing"
	"time"

	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/server/app/server_test"
	"github.com/buildbeaver/sdk/dynamic/bb"
	"github.com/stretchr/testify/require"
	"golang.org/x/net/context"
)

func TestDynamicBuildWorkflows(t *testing.T) {
	// This is a short test, no need to skip
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph := enqueueDynamicBuild(t, app, commit, nil)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	build := runDynamicBuildJobWorkflows(jobinatorEnv)
	defer build.Shutdown()
}

func runDynamicBuildJobWorkflows(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "BuildBeaver Dynamic Build Test Job for Workflow Names")

	var wg sync.WaitGroup
	wg.Add(2) // wait for all workflow functions to complete
	var baseJob, generateJob *bb.Job

	baseWorkflowFunc := func(workflow *bb.Workflow) error {
		defer wg.Done()
		// Make a basic job
		baseJob = bb.NewJob().
			Name("base-job").
			Desc("Base job description").
			Docker(bb.NewDocker().
				Image("docker:20.10").
				Pull(bb.DockerPullIfNotExists)).
			Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
			Step(bb.NewStep().
				Name("base-job-step").
				Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/"))
		workflow.Job(baseJob)

		// Make a second job with a job name that isn't unique between workflows
		workflow.Job(bb.NewJob().
			Name("second-job").
			Desc("A second job with a name that isn't unique between workflows").
			Docker(bb.NewDocker().
				Image("docker:20.10").
				Pull(bb.DockerPullIfNotExists)).
			Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
			Step(bb.NewStep().
				Name("second-job-step").
				Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/")))

		returnedJobs, err := workflow.Submit(false)
		require.NoError(t, err, "Error submitting new job(s) to build (1)")

		baseJobGraph := findJobByName(t.T, returnedJobs, baseJob.GetName())
		checkJobFromAPI(t.T, baseJob, baseJobGraph)
		require.Nil(t, baseJobGraph.Job.Error)
		return nil
	}

	generateWorkflowFunc := func(workflow *bb.Workflow) error {
		defer wg.Done()
		time.Sleep(1 * time.Second)

		// Make a job that depends on base-job
		generateJob = bb.NewJob().
			Name("generate-job").
			Desc("Generates all code").
			Type(bb.JobTypeDocker).
			Depends("base-workflow.base-job").
			Docker(bb.NewDocker().
				Image("buildbeaver/go-builder:latest").
				Pull(bb.DockerPullNever)).
			Step(bb.NewStep().
				Name("run-generate").
				Desc("This step does the actual generation of code").
				Commands("echo 'Hello world 2!'"))
		workflow.Job(generateJob)

		// Make a second job with a job name that isn't unique between workflows
		workflow.Job(bb.NewJob().
			Name("second-job").
			Desc("A second job with a name that isn't unique between workflows - inside generate workflow").
			Docker(bb.NewDocker().
				Image("docker:20.10").
				Pull(bb.DockerPullIfNotExists)).
			Fingerprint("sha1sum build/docker/go-builder/Dockerfile").
			Step(bb.NewStep().
				Name("second-job-generate-step").
				Commands("docker build -t buildbeaver/go-builder:latest build/docker/go-builder/")))

		returnedJobs, err := workflow.Submit(false)
		require.NoError(t, err, "Error submitting new job(s) to build (2)")

		generateJobGraph := findJobByName(t.T, returnedJobs, generateJob.GetName())
		checkJobFromAPI(t.T, generateJob, generateJobGraph)
		require.Nil(t, generateJobGraph.Job.Error)
		return nil
	}

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("base-workflow").Handler(baseWorkflowFunc),
		bb.NewWorkflow().Name("generate-workflow").Handler(generateWorkflowFunc),
	)
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK.")

	// Check the jobs went into the database correctly
	wg.Wait()
	checkJobCount(t, 5) // 2 jobs in each workflow + jobinator
	ctx := context.Background()
	buildFromServer, err := t.App.QueueService.ReadQueuedBuild(ctx, nil, t.BuildID)
	require.NoError(t, err)
	require.Equal(t, 5, len(buildFromServer.Jobs))
	jobMap := makeJobMap(buildFromServer.Jobs)
	baseJobFromServer := jobMap[jobReferenceToFQN(baseJob.GetReference())]
	require.NotNil(t, baseJobFromServer, "Could not find 'base' job on the server")
	generateJobFromServer := jobMap[jobReferenceToFQN(generateJob.GetReference())]
	require.NotNil(t, generateJobFromServer, "Could not find 'base' job on the server")

	checkJob(t.T, generateJob, generateJobFromServer)

	return build
}

func TestDynamicBuildWorkflowDuplicateNames(t *testing.T) {
	// This is a short test, no need to skip
	rand.Seed(time.Now().UnixNano())
	ctx := context.Background()

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph := enqueueDynamicBuild(t, app, commit, nil)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	build := runDynamicBuildJobWorkflowDuplicateNames(jobinatorEnv)
	if build != nil { // build should be nil if we failed to register our workflows
		build.Shutdown()
	}
}

func runDynamicBuildJobWorkflowDuplicateNames(t *JobinatorTestEnv) *bb.Build {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "BuildBeaver Dynamic Build Test Job for Workflow Duplicate Names")

	baseWorkflowFunc := func(workflow *bb.Workflow) error {
		bb.Log(bb.LogLevelInfo, "baseWorkflowFunc called")
		return nil
	}

	generateWorkflowFunc := func(workflow *bb.Workflow) error {
		bb.Log(bb.LogLevelInfo, "generateWorkflowFunc called")
		return nil
	}

	build, err := bb.WorkflowsWithEnv(t.Env, true,
		bb.NewWorkflow().Name("base-workflow").Handler(baseWorkflowFunc),
		bb.NewWorkflow().Name("base-workflow").Handler(generateWorkflowFunc),
	)
	require.Error(t, err, "Workflows with duplicate names should cause an error")

	return build
}

type workflowTestData struct {
	testName                  string
	workflows                 []*bb.WorkflowDefinition // handlers don't need to be specified and will be added later
	buildOptions              *models.BuildOptions
	expectedToRun             []bb.ResourceName
	expectedToNotRun          []bb.ResourceName
	expectedOrderDependencies map[bb.ResourceName]bb.ResourceName // maps dependent workflow name to workflow name that should already have been finished
}

var workflowDependencyTests = []workflowTestData{
	{
		testName: "Dependencies-test-1",
		workflows: []*bb.WorkflowDefinition{
			bb.NewWorkflow().Name("tests"),
			bb.NewWorkflow().
				Name("deploy").
				Depends("tests", bb.WorkflowConcurrent).
				Depends("spare-wheel", bb.WorkflowConcurrent),
			bb.NewWorkflow().
				Name("spare-wheel").
				Depends("round-peg", bb.WorkflowWait),
			bb.NewWorkflow().Name("round-peg"),
			bb.NewWorkflow().Name("square-pants"),
		},
		buildOptions: &models.BuildOptions{
			NodesToRun: []models.NodeFQN{models.NewNodeFQNForWorkflow("deploy")},
			Force:      false,
		},
		expectedToRun:    []bb.ResourceName{"tests", "deploy", "spare-wheel", "round-peg"},
		expectedToNotRun: []bb.ResourceName{"square-pants"},
		expectedOrderDependencies: map[bb.ResourceName]bb.ResourceName{
			"spare-wheel": "round-peg", // round-peg must finish before spare-wheel starts
		},
	},
	{
		testName: "Dependencies-test-2",
		workflows: []*bb.WorkflowDefinition{
			bb.NewWorkflow().Name("tests"),
			bb.NewWorkflow().
				Name("deploy").
				Depends("tests", bb.WorkflowConcurrent).
				Depends("spare-wheel", bb.WorkflowConcurrent),
			bb.NewWorkflow().
				Name("spare-wheel").
				Depends("round-peg", bb.WorkflowWait),
			bb.NewWorkflow().Name("round-peg"),
			bb.NewWorkflow().Name("square-pants"),
		},
		buildOptions: &models.BuildOptions{
			NodesToRun: []models.NodeFQN{models.NewNodeFQNForWorkflow("tests")},
			Force:      false,
		},
		expectedToRun:    []bb.ResourceName{"tests"},
		expectedToNotRun: []bb.ResourceName{"deploy", "spare-wheel", "round-peg", "square-pants"},
	},
	{
		testName: "Dependencies-test-3",
		workflows: []*bb.WorkflowDefinition{
			bb.NewWorkflow().Name("tests"),
			bb.NewWorkflow().
				Name("deploy").
				Depends("tests", bb.WorkflowConcurrent).
				Depends("spare-wheel", bb.WorkflowConcurrent),
			bb.NewWorkflow().
				Name("spare-wheel").
				Depends("round-peg", bb.WorkflowWait),
			bb.NewWorkflow().Name("round-peg"),
			bb.NewWorkflow().Name("square-pants"),
		},
		buildOptions: &models.BuildOptions{
			NodesToRun: []models.NodeFQN{}, // no nodes to run specified means 'run all workflows'
			Force:      false,
		},
		expectedToRun:    []bb.ResourceName{"tests", "deploy", "spare-wheel", "round-peg", "square-pants"},
		expectedToNotRun: []bb.ResourceName{},
		expectedOrderDependencies: map[bb.ResourceName]bb.ResourceName{
			"spare-wheel": "round-peg", // round-peg must finish before spare-wheel starts
		},
	},
}

func TestDynamicBuildWorkflowDependencies(t *testing.T) {
	rand.Seed(time.Now().UnixNano())

	for _, test := range workflowDependencyTests {
		runOneWorkflowTest(t, test)
	}
}

func runOneWorkflowTest(t *testing.T, testData workflowTestData) {
	ctx := context.Background()
	bb.Log(bb.LogLevelInfo, fmt.Sprintf("BuildBeaver Dynamic Build Workflow Test - running test '%s'", testData.testName))

	// Start a test server, listening on an arbitrary unused port
	app, cleanUpServer, err := server_test.New(server_test.TestConfig(t))
	require.Nil(t, err)
	defer cleanUpServer()
	app.CoreAPIServer.Start() // Start the HTTP server
	defer app.CoreAPIServer.Stop(ctx)
	commit, buildRunner := createPrerequisiteObjects(t, app)

	buildGraph := enqueueDynamicBuild(t, app, commit, testData.buildOptions)
	job, env := dequeueJob(t, app, buildRunner.ID, buildGraph.Jobs[0].Name)
	jobinatorEnv := NewJobinatorTestEnv(t, app, job, env)

	build, workflowsRun := runDynamicBuildJobWorkflowDependencies(jobinatorEnv, testData)
	build.Shutdown()

	// Check that the correct workflows were run
	for _, w := range testData.expectedToRun {
		require.True(t, workflowsRun[w], "Workflow '%s' should have been run but was not", w)
	}
	for _, w := range testData.expectedToNotRun {
		require.False(t, workflowsRun[w], "Workflow '%s' should NOT have been run but was", w)
	}
}

func runDynamicBuildJobWorkflowDependencies(
	t *JobinatorTestEnv,
	workflowData workflowTestData,
) (build *bb.Build, workflowsRun map[bb.ResourceName]bool) {
	bb.SetDefaultLogLevel(bb.LogLevelInfo)
	bb.Log(bb.LogLevelInfo, "BuildBeaver Dynamic Build Test Job for Workflow Dependencies")

	// Track which workflows got run
	var runMutex sync.Mutex
	run := make(map[bb.ResourceName]bool)
	markWorkflowAsRun := func(workflowName bb.ResourceName) {
		runMutex.Lock()
		defer runMutex.Unlock()
		run[workflowName] = true
	}

	// Handler function for workflows. If expectFinished is not empty then the handler will check that
	// the workflow with the specified name has already finished before this handler was called.
	workflowFunc := func(workflow *bb.Workflow, expectFinished bb.ResourceName) error {
		bb.Log(bb.LogLevelInfo, fmt.Sprintf("workflow function called for '%s'", workflow.GetName()))
		markWorkflowAsRun(workflow.GetName())

		// If this workflow was meant to start after another workflow, check that the other workflow has finished
		if expectFinished != "" {
			t.Logf("Checking that workflow %s has already finished now that %s is running", expectFinished, workflow.GetName())
			otherWorkflowFinished := workflow.IsWorkflowFinished(expectFinished)
			require.Truef(t, otherWorkflowFinished, "Workflow %s should be finished before workflow %s starts",
				expectFinished, workflow.GetName())
		}

		return nil
	}

	// Set the handler function for each workflow definition
	for _, w := range workflowData.workflows {
		// Pass expected order dependency in to the handler to check as the handler is run
		expectFinished := workflowData.expectedOrderDependencies[w.GetName()]

		w.Handler(func(workflow *bb.Workflow) error { return workflowFunc(workflow, expectFinished) })
	}

	build, err := bb.WorkflowsWithEnv(t.Env, true, workflowData.workflows...)
	require.NoError(t, err, "Error creating build workflows from env in dynamic API SDK.")

	t.Logf("Workflows to run: %v, workflows run %v", build.WorkflowsToRun, run)
	return build, run
}
