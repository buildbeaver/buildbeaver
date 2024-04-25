//go:build wireinject
// +build wireinject

package runner_test

import (
	"context"

	"github.com/benbjohnson/clock"
	"github.com/google/wire"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/common/models"
	"github.com/buildbeaver/buildbeaver/runner"
	"github.com/buildbeaver/buildbeaver/runner/app"
	"github.com/buildbeaver/buildbeaver/runner/logging"
	"github.com/buildbeaver/buildbeaver/server/api/rest/client"
)

func MakeLogPipelineFactory(
	client runner.APIClient,
	logFactory logger.LogFactory,
	runnerLogTempDir logging.RunnerLogTempDirectory,
) logging.LogPipelineFactory {
	return func(ctx context.Context, clk clock.Clock, secrets []*models.SecretPlaintext, logDescriptorID models.LogDescriptorID) (logging.LogPipeline, error) {
		return logging.NewClientLogPipeline(ctx, clk, logFactory, client, logDescriptorID, secrets, runnerLogTempDir, 0, 0, 0)
	}
}

func New(config *app.RunnerConfig) (*Runner, error) {
	panic(wire.Build(
		NewRunner,
		wire.FieldsOf(new(*app.RunnerConfig), "RunnerAPIEndpoints", "RunnerLogTempDir", "RunnerCertificateFile", "RunnerPrivateKeyFile", "AutoCreateCertificate", "CACertFile", "InsecureSkipVerify", "SchedulerConfig", "ExecutorConfig", "LogLevels"),
		client.NewClientCertificateAuthenticator,
		wire.Bind(new(client.Authenticator), new(*client.ClientCertificateAuthenticator)),
		client.NewAPIClient,
		wire.Bind(new(runner.APIClient), new(*client.APIClient)),
		runner.MakeExecutorFactory,
		runner.MakeOrchestratorFactory,
		runner.NewJobScheduler,
		logger.NewLogRegistry,
		logger.MakeLogrusLogFactoryStdOut,
		MakeLogPipelineFactory,
		runner.NewGitCheckoutManager,
	))
}
