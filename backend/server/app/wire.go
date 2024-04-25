//go:build wireinject
// +build wireinject

package app

import (
	"context"

	"github.com/benbjohnson/clock"
	"github.com/google/wire"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/artifact"
	"github.com/buildbeaver/buildbeaver/server/services/authentication"
	"github.com/buildbeaver/buildbeaver/server/services/authorization"
	"github.com/buildbeaver/buildbeaver/server/services/build"
	"github.com/buildbeaver/buildbeaver/server/services/credential"
	"github.com/buildbeaver/buildbeaver/server/services/encryption"
	"github.com/buildbeaver/buildbeaver/server/services/event"
	"github.com/buildbeaver/buildbeaver/server/services/group"
	"github.com/buildbeaver/buildbeaver/server/services/job"
	"github.com/buildbeaver/buildbeaver/server/services/keypair"
	"github.com/buildbeaver/buildbeaver/server/services/legal_entity"
	"github.com/buildbeaver/buildbeaver/server/services/log"
	"github.com/buildbeaver/buildbeaver/server/services/pull_request"
	"github.com/buildbeaver/buildbeaver/server/services/queue"
	"github.com/buildbeaver/buildbeaver/server/services/repo"
	"github.com/buildbeaver/buildbeaver/server/services/runner"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
	"github.com/buildbeaver/buildbeaver/server/services/scm/github"
	"github.com/buildbeaver/buildbeaver/server/services/secret"
	"github.com/buildbeaver/buildbeaver/server/services/step"
	"github.com/buildbeaver/buildbeaver/server/services/sync"
	"github.com/buildbeaver/buildbeaver/server/services/work_queue"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/artifacts"
	"github.com/buildbeaver/buildbeaver/server/store/authorizations"
	"github.com/buildbeaver/buildbeaver/server/store/builds"
	"github.com/buildbeaver/buildbeaver/server/store/commits"
	"github.com/buildbeaver/buildbeaver/server/store/credentials"
	"github.com/buildbeaver/buildbeaver/server/store/events"
	"github.com/buildbeaver/buildbeaver/server/store/grants"
	"github.com/buildbeaver/buildbeaver/server/store/group_memberships"
	"github.com/buildbeaver/buildbeaver/server/store/groups"
	"github.com/buildbeaver/buildbeaver/server/store/identities"
	"github.com/buildbeaver/buildbeaver/server/store/jobs"
	"github.com/buildbeaver/buildbeaver/server/store/legal_entities"
	"github.com/buildbeaver/buildbeaver/server/store/legal_entity_memberships"
	"github.com/buildbeaver/buildbeaver/server/store/logs"
	"github.com/buildbeaver/buildbeaver/server/store/migrations"
	"github.com/buildbeaver/buildbeaver/server/store/ownerships"
	"github.com/buildbeaver/buildbeaver/server/store/pull_requests"
	"github.com/buildbeaver/buildbeaver/server/store/repos"
	"github.com/buildbeaver/buildbeaver/server/store/resource_links"
	"github.com/buildbeaver/buildbeaver/server/store/runners"
	"github.com/buildbeaver/buildbeaver/server/store/secrets"
	"github.com/buildbeaver/buildbeaver/server/store/steps"
	"github.com/buildbeaver/buildbeaver/server/store/work_item_states"
	"github.com/buildbeaver/buildbeaver/server/store/work_items"
)

func MakeSCMs(
	scmRegistry *scm.SCMRegistry,
	db *store.DB,
	repoStore store.RepoStore,
	commitStore store.CommitStore,
	buildStore store.BuildStore,
	pullRequestService services.PullRequestService,
	legalEntityService services.LegalEntityService,
	queueService services.QueueService,
	workQueueService services.WorkQueueService,
	groupService services.GroupService,
	syncService services.SyncService,
	githubServiceConfig github.AppConfig,
	logFactory logger.LogFactory,
) []scm.SCM {
	github := github.NewGitHubService(
		db,
		repoStore,
		commitStore,
		buildStore,
		pullRequestService,
		legalEntityService,
		queueService,
		workQueueService,
		groupService,
		syncService,
		githubServiceConfig,
		logFactory)
	scmRegistry.Register(github)

	return []scm.SCM{
		github,
	}
}

// MakeWorkQueueService creates a new instance of WorkQueueService and calls Start() to begin
// processing work items from the queue.
func MakeWorkQueueService(
	db *store.DB,
	workItemStore store.WorkItemStore,
	stateStore store.WorkItemStateStore,
	logFactory logger.LogFactory,
) *work_queue.WorkQueueService {
	service := work_queue.NewWorkQueueService(db, workItemStore, stateStore, logFactory)
	service.Start()
	return service
}

func New(ctx context.Context, config *ServerConfig) (*Server, func(), error) {
	panic(wire.Build(
		NewServer,
		wire.FieldsOf(new(*ServerConfig), "BlobStoreConfig", "EncryptionConfig", "CoreAPIConfig", "RunnerAPIConfig", "InternalRunnerConfig", "AuthenticationConfig", "DatabaseConfig", "GitHubAppConfig", "LogLevels", "LogServiceConfig", "JWTConfig", "LimitsConfig"),
		scm.NewSCMRegistry,
		store.NewDatabase,
		migrations.NewBBGolangMigrateRunner,
		wire.Bind(new(store.MigrationRunner), new(*migrations.GolangMigrateRunner)),

		// Stores
		repos.NewStore,
		wire.Bind(new(store.RepoStore), new(*repos.RepoStore)),
		commits.NewStore,
		wire.Bind(new(store.CommitStore), new(*commits.CommitStore)),
		builds.NewStore,
		wire.Bind(new(store.BuildStore), new(*builds.BuildStore)),
		jobs.NewStore,
		wire.Bind(new(store.JobStore), new(*jobs.JobStore)),
		steps.NewStore,
		wire.Bind(new(store.StepStore), new(*steps.StepStore)),
		secrets.NewStore,
		wire.Bind(new(store.SecretStore), new(*secrets.SecretStore)),
		ownerships.NewStore,
		wire.Bind(new(store.OwnershipStore), new(*ownerships.OwnershipStore)),
		legal_entities.NewStore,
		wire.Bind(new(store.LegalEntityStore), new(*legal_entities.LegalEntityStore)),
		legal_entity_memberships.NewStore,
		wire.Bind(new(store.LegalEntityMembershipStore), new(*legal_entity_memberships.LegalEntityMembershipStore)),
		identities.NewStore,
		wire.Bind(new(store.IdentityStore), new(*identities.IdentityStore)),
		authorizations.NewStore,
		wire.Bind(new(store.AuthorizationStore), new(*authorizations.AuthorizationStore)),
		credentials.NewStore,
		wire.Bind(new(store.CredentialStore), new(*credentials.CredentialStore)),
		groups.NewStore,
		wire.Bind(new(store.GroupStore), new(*groups.GroupStore)),
		group_memberships.NewStore,
		wire.Bind(new(store.GroupMembershipStore), new(*group_memberships.GroupMembershipStore)),
		grants.NewStore,
		wire.Bind(new(store.GrantStore), new(*grants.GrantStore)),
		artifacts.NewStore,
		wire.Bind(new(store.ArtifactStore), new(*artifacts.ArtifactStore)),
		runners.NewStore,
		wire.Bind(new(store.RunnerStore), new(*runners.RunnerStore)),
		resource_links.NewStore,
		wire.Bind(new(store.ResourceLinkStore), new(*resource_links.ResourceLinkStore)),
		logs.NewStore,
		wire.Bind(new(store.LogStore), new(*logs.LogStore)),
		pull_requests.NewStore,
		wire.Bind(new(store.PullRequestStore), new(*pull_requests.PullRequestStore)),
		work_items.NewStore,
		wire.Bind(new(store.WorkItemStore), new(*work_items.WorkItemStore)),
		work_item_states.NewStore,
		wire.Bind(new(store.WorkItemStateStore), new(*work_item_states.WorkItemStateStore)),
		events.NewStore,
		wire.Bind(new(store.EventStore), new(*events.EventStore)),

		// Services
		queue.NewQueueService,
		wire.Bind(new(services.QueueService), new(*queue.QueueService)),
		log.NewLogService,
		wire.Bind(new(services.LogService), new(*log.LogService)),
		encryption.NewEncryptionService,
		wire.Bind(new(services.EncryptionService), new(*encryption.EncryptionService)),
		secret.NewSecretService,
		wire.Bind(new(services.SecretService), new(*secret.SecretService)),
		authorization.NewAuthorizationService,
		wire.Bind(new(services.AuthorizationService), new(*authorization.AuthorizationService)),
		group.NewGroupService,
		wire.Bind(new(services.GroupService), new(*group.GroupService)),
		authentication.NewAuthenticationService,
		wire.Bind(new(services.AuthenticationService), new(*authentication.AuthenticationService)),
		credential.NewCredentialService,
		wire.Bind(new(services.CredentialService), new(*credential.CredentialService)),
		artifact.NewArtifactService,
		wire.Bind(new(services.ArtifactService), new(*artifact.ArtifactService)),
		legal_entity.NewLegalEntityService,
		wire.Bind(new(services.LegalEntityService), new(*legal_entity.LegalEntityService)),
		repo.NewRepoService,
		wire.Bind(new(services.RepoService), new(*repo.RepoService)),
		pull_request.NewPullRequestService,
		wire.Bind(new(services.PullRequestService), new(*pull_request.PullRequestService)),
		keypair.NewKeyPairService,
		wire.Bind(new(services.KeyPairService), new(*keypair.KeyPairService)),
		build.NewBuildService,
		wire.Bind(new(services.BuildService), new(*build.BuildService)),
		job.NewJobService,
		wire.Bind(new(services.JobService), new(*job.JobService)),
		step.NewStepService,
		wire.Bind(new(services.StepService), new(*step.StepService)),
		sync.NewSyncService,
		wire.Bind(new(services.SyncService), new(*sync.SyncService)),
		runner.NewRunnerService,
		wire.Bind(new(services.RunnerService), new(*runner.RunnerService)),
		MakeWorkQueueService,
		wire.Bind(new(services.WorkQueueService), new(*work_queue.WorkQueueService)),
		event.NewEventService,
		wire.Bind(new(services.EventService), new(*event.EventService)),

		BlobStoreFactory,
		KeyManagerFactory,

		// APIs
		routes.NewResourceLinker,
		server.NewLogAPI,
		server.NewQueueAPI,
		server.NewWebhooksAPI,
		server.NewSecretAPI,
		server.NewCoreAuthenticationAPI,
		server.NewArtifactAPI,
		server.NewRootAPI,
		server.NewLegalEntityAPI,
		server.NewRepoAPI,
		server.NewBuildAPI,
		server.NewRunnerAPI,
		server.NewJobAPI,
		server.NewDynamicJobAPI,
		server.NewStepAPI,
		server.NewSearchAPI,
		server.NewTokenExchangeAPI,

		// HTTP Servers
		server.NewAppAPIServer,
		server.NewAppAPIRouter,
		server.NewRunnerAPIServer,
		server.NewRunnerAPIRouter,
		server.RealHTTPServerFactory,

		MakeSCMs,
		NewInternalRunnerManager,
		logger.NewLogRegistry,
		logger.MakeLogrusLogFactoryStdOut,
		clock.New,
	))
}
