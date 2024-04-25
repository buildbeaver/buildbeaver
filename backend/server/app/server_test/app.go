package server_test

import (
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type TestServer struct {
	DB                         *store.DB
	SCMRegistry                *scm.SCMRegistry
	ArtifactStore              store.ArtifactStore
	RepoStore                  store.RepoStore
	CommitStore                store.CommitStore
	BuildStore                 store.BuildStore
	BuildService               services.BuildService
	SecretStore                store.SecretStore
	JobService                 services.JobService
	JobStore                   store.JobStore
	StepStore                  store.StepStore
	LegalEntityStore           store.LegalEntityStore
	LegalEntityMembershipStore store.LegalEntityMembershipStore
	IdentityStore              store.IdentityStore
	GroupStore                 store.GroupStore
	GroupMembershipStore       store.GroupMembershipStore
	GrantStore                 store.GrantStore
	OwnershipStore             store.OwnershipStore
	CredentialStore            store.CredentialStore
	AuthorizationStore         store.AuthorizationStore
	ResourceLinkStore          store.ResourceLinkStore
	EventStore                 store.EventStore
	LogStore                   store.LogStore
	LogService                 services.LogService
	PullRequestStore           store.PullRequestStore
	RunnerService              services.RunnerService
	QueueService               services.QueueService
	CredentialService          services.CredentialService
	LegalEntityService         services.LegalEntityService
	AuthorizationService       services.AuthorizationService
	GroupService               services.GroupService
	PullRequestService         services.PullRequestService
	RepoService                services.RepoService
	StepService                services.StepService
	SyncService                services.SyncService
	WorkItemStore              store.WorkItemStore
	WorkItemStateStore         store.WorkItemStateStore
	WorkQueueService           services.WorkQueueService
	EventService               services.EventService
	ArtifactService            services.ArtifactService
	LogFactory                 logger.LogFactory

	CoreAPIServer   *server.AppAPIServer
	RunnerAPIServer *server.RunnerAPIServer
}

func NewTestServer(
	db *store.DB,
	scmRegistry *scm.SCMRegistry,
	artifactStore store.ArtifactStore,
	repoStore store.RepoStore,
	commitStore store.CommitStore,
	buildStore store.BuildStore,
	buildService services.BuildService,
	secretStore store.SecretStore,
	jobService services.JobService,
	jobStore store.JobStore,
	stepStore store.StepStore,
	legalEntityStore store.LegalEntityStore,
	legalEntityMembershipStore store.LegalEntityMembershipStore,
	identityStore store.IdentityStore,
	groupStore store.GroupStore,
	groupMembershipStore store.GroupMembershipStore,
	grantStore store.GrantStore,
	ownershipStore store.OwnershipStore,
	credentialStore store.CredentialStore,
	authorizationStore store.AuthorizationStore,
	resourceLinkStore store.ResourceLinkStore,
	eventStore store.EventStore,
	logStore store.LogStore,
	logService services.LogService,
	pullRequestStore store.PullRequestStore,
	runnerService services.RunnerService,
	queueService services.QueueService,
	credentialService services.CredentialService,
	legalEntityService services.LegalEntityService,
	authorizationService services.AuthorizationService,
	groupService services.GroupService,
	pullRequestService services.PullRequestService,
	repoService services.RepoService,
	stepService services.StepService,
	syncService services.SyncService,
	workItemStore store.WorkItemStore,
	workItemStateStore store.WorkItemStateStore,
	workQueueService services.WorkQueueService,
	eventService services.EventService,
	artifactService services.ArtifactService,
	logFactory logger.LogFactory,
	coreAPIServer *server.AppAPIServer,
	runnerAPIServer *server.RunnerAPIServer,
	allSCMs []scm.SCM, // tell Wire the app has a dependency on the SCMs, to ensure they're created
) *TestServer {
	return &TestServer{
		DB:                         db,
		SCMRegistry:                scmRegistry,
		ArtifactStore:              artifactStore,
		RepoStore:                  repoStore,
		CommitStore:                commitStore,
		BuildStore:                 buildStore,
		BuildService:               buildService,
		SecretStore:                secretStore,
		JobService:                 jobService,
		JobStore:                   jobStore,
		StepStore:                  stepStore,
		LegalEntityStore:           legalEntityStore,
		LegalEntityMembershipStore: legalEntityMembershipStore,
		IdentityStore:              identityStore,
		GroupStore:                 groupStore,
		GroupMembershipStore:       groupMembershipStore,
		GrantStore:                 grantStore,
		OwnershipStore:             ownershipStore,
		CredentialStore:            credentialStore,
		AuthorizationStore:         authorizationStore,
		ResourceLinkStore:          resourceLinkStore,
		EventStore:                 eventStore,
		LogStore:                   logStore,
		LogService:                 logService,
		PullRequestStore:           pullRequestStore,
		RunnerService:              runnerService,
		QueueService:               queueService,
		CredentialService:          credentialService,
		LegalEntityService:         legalEntityService,
		AuthorizationService:       authorizationService,
		GroupService:               groupService,
		PullRequestService:         pullRequestService,
		RepoService:                repoService,
		StepService:                stepService,
		SyncService:                syncService,
		WorkItemStore:              workItemStore,
		WorkItemStateStore:         workItemStateStore,
		WorkQueueService:           workQueueService,
		EventService:               eventService,
		ArtifactService:            artifactService,
		LogFactory:                 logFactory,
		CoreAPIServer:              coreAPIServer,
		RunnerAPIServer:            runnerAPIServer,
	}
}
