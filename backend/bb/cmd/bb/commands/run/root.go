package run

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/buildbeaver/buildbeaver/bb/app"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/commands"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/utils"
	"github.com/buildbeaver/buildbeaver/common/models"
)

func init() {
	runRootCmd.PersistentFlags().StringVar(
		&runCmdConfig.workDir,
		"workdir",
		"~/.bb/local",
		"The scratch space to use for local builds")
	runRootCmd.PersistentFlags().BoolVarP(
		&runCmdConfig.verbose,
		"verbose",
		"v",
		false,
		"Enable verbose log output")
	runRootCmd.PersistentFlags().BoolVarP(
		&runCmdConfig.force,
		"force",
		"f",
		false,
		"Force all jobs to re-run by ignoring fingerprints")
	runRootCmd.PersistentFlags().BoolVar(
		&runCmdConfig.skipCleanup,
		"skip-cleanup",
		false,
		"Do not attempt to clean up resources (including docker containers and networks) left over from previous runs")
	commands.RootCmd.AddCommand(runRootCmd)
}

var runCmdConfig = struct {
	workDir     string
	verbose     bool
	force       bool
	skipCleanup bool
}{}

var runRootCmd = &cobra.Command{
	Use:           "run [workflow]...",
	Short:         "Run one or more build jobs",
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		var (
			err error
			ctx = context.Background()
		)

		lockFile, err := utils.GetBBFileLock()
		if err != nil {
			return errors.Wrap(err, "Error: Another instance of BB is currently running")
		}
		defer lockFile.Close()

		runCmdConfig.workDir, err = utils.HomeifyPath(runCmdConfig.workDir)
		if err != nil {
			return err
		}

		err = os.MkdirAll(runCmdConfig.workDir, 0770)
		if err != nil {
			return fmt.Errorf("error making work directory %q: %w", runCmdConfig.workDir, err)
		}

		config := app.NewBBConfig(runCmdConfig.workDir, runCmdConfig.verbose, commands.Global.JSON)

		// Clear out all old blobs - they don't need to persist between runs
		os.Remove(config.LocalBlobStoreDir.String())

		bb, cleanup, err := app.New(ctx, config)
		if err != nil {
			// The local sqlite database is effectively a cache. Blow it away at the first
			// sign of trouble and try again.
			os.Remove(config.DatabaseFilePath)
			bb, cleanup, err = app.New(ctx, config)
			if err != nil {
				return errors.Wrap(err, "error initializing app")
			}
		}
		defer cleanup()

		if !runCmdConfig.skipCleanup {
			utils.CleanUpOldResources(bb, runCmdConfig.verbose)
		}

		err = bb.Backend.Start()
		if err != nil {
			return errors.Wrap(err, "error starting backend")
		}
		defer bb.Backend.Stop()

		bb.APIServer.Start()

		fqns, err := utils.ParseNodeFQNS(args)
		if err != nil {
			return fmt.Errorf("error parsing steps: %v", err)
		}
		opts := &models.BuildOptions{NodesToRun: fqns, Force: runCmdConfig.force}

		build, err := bb.Backend.Enqueue(ctx, opts)
		if err != nil {
			return fmt.Errorf("error queuing local build: %v", err)
		}

		bb.JobScheduler.Start()
		// HACK wait some time to allow the scheduler to try pick up a job
		// before we call StopWhenQuiet
		for i := 0; i < 10; i++ {
			stats := bb.JobScheduler.GetStats()
			if stats.FailedPollCount == 0 && stats.SuccessfulPollCount == 0 {
				time.Sleep(time.Millisecond * 100)
			}
		}
		bb.JobScheduler.StopWhenQuiet()

		failedJobs := bb.Backend.Results()

		if !config.Verbose {
			if len(failedJobs) > 0 {
				fmt.Fprint(os.Stdout, "\r\n")
				fmt.Fprintf(os.Stdout, "%d job(s) failed. See logs for details.\r\n\r\n", len(failedJobs))
				t := true
				reader, _ := bb.LogService.ReadData(ctx, build.LogDescriptorID, &models.LogSearch{
					Plaintext: &t,
					Expand:    &t,
				})
				io.Copy(os.Stdout, reader)
			}
		}
		if len(failedJobs) > 0 {
			os.Exit(1)
		} else {
			os.Exit(0)
		}
		return nil
	},
}
