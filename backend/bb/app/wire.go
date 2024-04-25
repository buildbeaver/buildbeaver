//go:build wireinject
// +build wireinject

package app

import (
	"context"

	"github.com/benbjohnson/clock"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/local_backend"
	"github.com/google/wire"

	"github.com/buildbeaver/buildbeaver/bb/bb_server"
	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	runner2 "github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/routes"
	"github.com/buildbeaver/buildbeaver/server/api/rest/server"
	"github.com/buildbeaver/buildbeaver/server/services"
	"github.com/buildbeaver/buildbeaver/server/services/artifact"
	"github.com/buildbeaver/buildbeaver/server/services/authentication"
	"github.com/buildbeaver/buildbeaver/server/services/authorization"
	"github.com/buildbeaver/buildbeaver/server/services/blob"
	"github.com/buildbeaver/buildbeaver/server/services/build"
	"github.com/buildbeaver/buildbeaver/server/services/credential"
	"github.com/buildbeaver/buildbeaver/server/services/encryption"
	"github.com/buildbeaver/buildbeaver/server/services/event"
	"github.com/buildbeaver/buildbeaver/server/services/group"
	"github.com/buildbeaver/buildbeaver/server/services/job"
	"github.com/buildbeaver/buildbeaver/server/services/keypair"
	"github.com/buildbeaver/buildbeaver/server/services/legal_entity"
	"github.com/buildbeaver/buildbeaver/server/services/log"
	"github.com/buildbeaver/buildbeaver/server/services/queue"
	"github.com/buildbeaver/buildbeaver/server/services/repo"
	"github.com/buildbeaver/buildbeaver/server/services/runner"
	"github.com/buildbeaver/buildbeaver/server/services/scm"
	"github.com/buildbeaver/buildbeaver/server/services/secret"
	"github.com/buildbeaver/buildbeaver/server/services/step"
	"github.com/buildbeaver/buildbeaver/server/services/sync"
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
	"github.com/buildbeaver/buildbeaver/server/store/repos"
	"github.com/buildbeaver/buildbeaver/server/store/resource_links"
	"github.com/buildbeaver/buildbeaver/server/store/runners"
	"github.com/buildbeaver/buildbeaver/server/store/secrets"
	"github.com/buildbeaver/buildbeaver/server/store/steps"
)

func MakeLogPipelineFactory(
	client runner2.APIClient,
	logFactory logger.LogFactory,
	runnerLogTempDir logging.RunnerLogTempDirectory,
) logging.LogPipelineFactory {
	return func(ctx context.Context, clk clock.Clock, secrets []*models.SecretPlaintext, logDescriptorID models.LogDescriptorID) (logging.LogPipeline, error) {
		return logging.NewClientLogPipeline(ctx, clk, logFactory, client, logDescriptorID, secrets, runnerLogTempDir, 0, 0, 0)
	}
}

func New(ctx context.Context, config *BBConfig) (*App, func(), error) {
	panic(wire.Build(
		wire.Struct(new(App), "*"),
		wire.Struct(new(local_backend.LocalBackendConfig), "*"),
		local_backend.NewLocalBackend,
		wire.FieldsOf(new(*BBConfig), "BBAPIConfig", "LocalBlobStoreDir", "LogFilePath", "LocalKeyManagerMasterKey", "DatabaseConfig", "RunnerLogTempDir", "SchedulerConfig", "ExecutorConfig", "LogLevels", "LogServiceConfig", "JWTConfig", "LimitsConfig", "JSON", "Verbose"),
		store.NewDatabase,
		migrations.NewBBGolangMigrateRunner,
		wire.Bind(new(store.MigrationRunner), new(*migrations.GolangMigrateRunner)),
		scm.NewSCMRegistry, // can stay empty but required for some components

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
		artifacts.NewStore,
		wire.Bind(new(store.ArtifactStore), new(*artifacts.ArtifactStore)),
		runners.NewStore,
		wire.Bind(new(store.RunnerStore), new(*runners.RunnerStore)),
		credentials.NewStore,
		wire.Bind(new(store.CredentialStore), new(*credentials.CredentialStore)),
		groups.NewStore,
		wire.Bind(new(store.GroupStore), new(*groups.GroupStore)),
		group_memberships.NewStore,
		wire.Bind(new(store.GroupMembershipStore), new(*group_memberships.GroupMembershipStore)),
		grants.NewStore,
		wire.Bind(new(store.GrantStore), new(*grants.GrantStore)),
		resource_links.NewStore,
		wire.Bind(new(store.ResourceLinkStore), new(*resource_links.ResourceLinkStore)),
		logs.NewStore,
		wire.Bind(new(store.LogStore), new(*logs.LogStore)),
		events.NewStore,
		wire.Bind(new(store.EventStore), new(*events.EventStore)),

		// Services
		queue.NewQueueService,
		wire.Bind(new(services.QueueService), new(*queue.QueueService)),
		legal_entity.NewLegalEntityService,
		wire.Bind(new(services.LegalEntityService), new(*legal_entity.LegalEntityService)),
		log.NewLogService,
		wire.Bind(new(services.LogService), new(*log.LogService)),
		encryption.NewEncryptionService,
		wire.Bind(new(services.EncryptionService), new(*encryption.EncryptionService)),
		secret.NewSecretService,
		wire.Bind(new(services.SecretService), new(*secret.SecretService)),
		credential.NewCredentialService,
		repo.NewRepoService,
		wire.Bind(new(services.RepoService), new(*repo.RepoService)),
		keypair.NewKeyPairService,
		wire.Bind(new(services.KeyPairService), new(*keypair.KeyPairService)),
		wire.Bind(new(services.CredentialService), new(*credential.CredentialService)),
		artifact.NewArtifactService,
		wire.Bind(new(services.ArtifactService), new(*artifact.ArtifactService)),
		wire.Bind(new(runner2.APIClient), new(*local_backend.LocalBackend)),
		runner2.NewJobScheduler,
		build.NewBuildService,
		wire.Bind(new(services.BuildService), new(*build.BuildService)),
		job.NewJobService,
		wire.Bind(new(services.JobService), new(*job.JobService)),
		step.NewStepService,
		wire.Bind(new(services.StepService), new(*step.StepService)),
		group.NewGroupService,
		wire.Bind(new(services.GroupService), new(*group.GroupService)),
		authorization.NewAuthorizationService,
		wire.Bind(new(services.AuthorizationService), new(*authorization.AuthorizationService)),
		runner.NewRunnerService,
		wire.Bind(new(services.RunnerService), new(*runner.RunnerService)),
		event.NewEventService,
		wire.Bind(new(services.EventService), new(*event.EventService)),
		authentication.NewAuthenticationService,
		wire.Bind(new(services.AuthenticationService), new(*authentication.AuthenticationService)),
		// TODO: Can we not use sync service? Needed by AuthenticationService for GitHub OAuth authentication, which we don't need in bb
		sync.NewSyncService,
		wire.Bind(new(services.SyncService), new(*sync.SyncService)),

		blob.NewLocalBlobStore,
		wire.Bind(new(services.BlobStore), new(*blob.LocalBlobStore)),
		encryption.NewLocalKeyManager,
		wire.Bind(new(encryption.KeyManager), new(*encryption.LocalKeyManager)),

		// APIs
		routes.NewResourceLinker,
		server.NewLogAPI,
		wire.Bind(new(server.ArtifactAPIDynamic), new(*bb_server.ArtifactAPIProxy)),
		server.NewArtifactAPI,
		bb_server.NewArtifactAPIProxy,
		server.NewRootAPI,
		server.NewBuildAPI,
		server.NewJobAPI,
		wire.Bind(new(server.DynamicJobAPIDynamic), new(*bb_server.DynamicJobAPIProxy)),
		server.NewDynamicJobAPI,
		bb_server.NewDynamicJobAPIProxy, // proxy the real dynamic API to inject code

		// HTTP Servers
		bb_server.NewBBAPIServer,
		bb_server.NewBBAPIRouter,
		server.RealHTTPServerFactory,

		// Built-in runner
		runner2.MakeExecutorFactory,
		runner2.MakeOrchestratorFactory,
		MakeLogPipelineFactory,
		runner2.NewGitCheckoutManager,

		logger.NewLogRegistry,
		logger.MakeLogrusLogFactoryToFile,
		clock.New))
}
