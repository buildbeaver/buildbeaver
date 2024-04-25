package cleanup

import (
	"os"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"golang.org/x/net/context"

	"github.com/buildbeaver/buildbeaver/bb/app"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/commands"
	"github.com/buildbeaver/buildbeaver/bb/cmd/bb/utils"
)

func init() {
	cleanupRootCmd.PersistentFlags().StringVar(
		&cleanupCmdConfig.workDir,
		"workdir",
		"~/.bb/local",
		"The scratch space to use for local builds")
	cleanupRootCmd.PersistentFlags().BoolVarP(
		&cleanupCmdConfig.verbose,
		"verbose",
		"v",
		false,
		"Enable verbose log output")
	commands.RootCmd.AddCommand(cleanupRootCmd)
}

var cleanupCmdConfig = struct {
	workDir string
	verbose bool
}{}

var cleanupRootCmd = &cobra.Command{
	Use:           "cleanup",
	Short:         "Clean up resources (including docker containers and networks) left over from previous runs",
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

		cleanupCmdConfig.workDir, err = utils.HomeifyPath(cleanupCmdConfig.workDir)
		if err != nil {
			return err
		}

		config := app.NewBBConfig(cleanupCmdConfig.workDir, cleanupCmdConfig.verbose, false)

		// Clear out all old blobs
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

		utils.CleanUpOldResources(bb, true)

		return nil
	},
}
