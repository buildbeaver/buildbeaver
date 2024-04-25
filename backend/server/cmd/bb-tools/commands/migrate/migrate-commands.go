package migrate

import (
	"context"
	"fmt"
	"strconv"

	"github.com/spf13/cobra"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/cli"
	"github.com/buildbeaver/buildbeaver/server/cmd/bb-tools/commands"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/migrations"
)

const defaultSQLiteConnectionString = "file:/var/lib/buildbeaver/db/sqlite.db?cache=shared"

func init() {
	migrateRootCmd.PersistentFlags().StringVar(
		&migrateCmdConfig.databaseDriver,
		"driver",
		string(store.Sqlite),
		"The Database Driver to use for migration (i.e sqlite3|postgres)")
	migrateRootCmd.PersistentFlags().StringVar(
		&migrateCmdConfig.databaseConnectionString,
		"connection",
		defaultSQLiteConnectionString,
		"The connection string for the database to use for migration")
	migrateRootCmd.PersistentFlags().BoolVarP(
		&migrateCmdConfig.verbose,
		"verbose",
		"v",
		false,
		"Enable verbose log output")
	migrateRootCmd.PersistentFlags().BoolVarP(
		&migrateCmdConfig.skipConfirmation,
		"skip-confirmation",
		"",
		false,
		"Skip interactive confirmation and automatically answer Yes to confirmation questions")

	commands.RootCmd.AddCommand(migrateRootCmd)
	migrateRootCmd.AddCommand(migrateUpCmd)
	migrateRootCmd.AddCommand(migrateDownCmd)
	migrateRootCmd.AddCommand(migrateGotoCmd)
	migrateRootCmd.AddCommand(migrateForceCmd)
}

var migrateCmdConfig = struct {
	databaseDriver           string
	databaseConnectionString string
	verbose                  bool
	skipConfirmation         bool
	migrationRunner          store.MigrationRunner
}{}

var migrateRootCmd = &cobra.Command{
	Use:   "migrate up|down|goto version-number",
	Short: "Migrates the database up to the latest version, down to empty, or to a specific version number",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		// migration runner needs a log factory; use a very plain log format
		logRegistry, err := logger.NewLogRegistry("")
		if err != nil {
			return err
		}
		logFactory := logger.MakeLogrusLogFactoryStdOutPlain(logRegistry)

		migrateCmdConfig.migrationRunner = migrations.NewBBGolangMigrateRunner(logFactory)
		return nil
	},
}

var migrateUpCmd = &cobra.Command{
	Use:           "up",
	Short:         "Migrates the database up to the latest version",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		err := migrateCmdConfig.migrationRunner.Up(
			context.Background(),
			store.DBDriver(migrateCmdConfig.databaseDriver),
			store.DatabaseConnectionString(migrateCmdConfig.databaseConnectionString),
		)
		if err != nil {
			return fmt.Errorf("error running 'up' migration: %w", err)
		}
		return nil
	},
}

var migrateDownCmd = &cobra.Command{
	Use:           "down",
	Short:         "Migrates the database down to being empty",
	Args:          cobra.NoArgs,
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		confirmed := cli.AskForConfirmation("Running a Down migration will remove ALL data from this database. Are you sure?", migrateCmdConfig.skipConfirmation)
		if confirmed {
			err := migrateCmdConfig.migrationRunner.Down(
				context.Background(),
				store.DBDriver(migrateCmdConfig.databaseDriver),
				store.DatabaseConnectionString(migrateCmdConfig.databaseConnectionString),
			)
			if err != nil {
				return fmt.Errorf("error running 'down' migration: %w", err)
			}
		} else {
			cli.Stdout.Printf("Down migration cancelled.")
		}
		return nil
	},
}

var migrateGotoCmd = &cobra.Command{
	Use:           "goto V",
	Short:         "Migrates the database up or down as required to be at specific version V",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.Atoi(args[0])
		if err != nil || version <= 0 {
			return fmt.Errorf("error: version must be a valid number")
		}

		confirmed := cli.AskForConfirmation("Running a Goto migration will sometimes REMOVE data from this database. Are you sure?", migrateCmdConfig.skipConfirmation)
		if confirmed {
			err = migrateCmdConfig.migrationRunner.Goto(
				context.Background(),
				store.DBDriver(migrateCmdConfig.databaseDriver),
				store.DatabaseConnectionString(migrateCmdConfig.databaseConnectionString),
				uint(version),
			)
			if err != nil {
				return fmt.Errorf("error running 'goto' migration: %w", err)
			}
		} else {
			cli.Stdout.Printf("Goto migration cancelled.")
		}
		return nil
	},
}

var migrateForceCmd = &cobra.Command{
	Use:           "force V",
	Short:         "Marks the database as being clean and in version V, but don't run migrations",
	Args:          cobra.ExactArgs(1),
	SilenceUsage:  true,
	SilenceErrors: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		version, err := strconv.Atoi(args[0])
		if err != nil || version <= 0 {
			return fmt.Errorf("error: version must be a valid number")
		}

		confirmed := cli.AskForConfirmation("Running a Force migration should only be performed after the database has been manually checked and fixed. Are you sure?", migrateCmdConfig.skipConfirmation)
		if confirmed {
			err = migrateCmdConfig.migrationRunner.Force(
				context.Background(),
				store.DBDriver(migrateCmdConfig.databaseDriver),
				store.DatabaseConnectionString(migrateCmdConfig.databaseConnectionString),
				uint(version))
			if err != nil {
				return fmt.Errorf("error running 'force' operation: %w", err)
			}
		} else {
			cli.Stdout.Printf("Force migration cancelled.")
		}
		return nil
	},
}
