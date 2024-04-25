package migrations

import (
	"bytes"
	"context"
	"fmt"
	"text/template"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migrate_database "github.com/golang-migrate/migrate/v4/database"
	migrate_postgres "github.com/golang-migrate/migrate/v4/database/postgres"
	migrate_sqlite3 "github.com/golang-migrate/migrate/v4/database/sqlite3"
	migrate_iofs "github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/jmoiron/sqlx"
	"github.com/psanford/memfs"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/store"
)

type GolangMigrateRunner struct {
	migrationData MigrationSet
	logger.Log
}

// NewGolangMigrateRunner creates a migration runner using the golang-migrate library to perform the migrations
// specified in migrationData.
func NewGolangMigrateRunner(
	migrationData MigrationSet,
	logFactory logger.LogFactory,
) *GolangMigrateRunner {
	return &GolangMigrateRunner{
		migrationData: migrationData,
		Log:           logFactory("GolangMigrateRunner"),
	}
}

// NewBBGolangMigrateRunner creates a migration runner using the golang-migrate library to perform the standard
// set of migrations for the BuildBeaver server database.
func NewBBGolangMigrateRunner(logFactory logger.LogFactory) *GolangMigrateRunner {
	return NewGolangMigrateRunner(BuildBeaverServerMigrations, logFactory)
}

func (r *GolangMigrateRunner) Up(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString) error {
	return r.runMigrationFunction(ctx, driver, connectionString, func(migrator *migrate.Migrate) error {
		r.Infof("Running migrations up to latest database version...")
		return migrator.Up()
	})
}

func (r *GolangMigrateRunner) Down(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString) error {
	return r.runMigrationFunction(ctx, driver, connectionString, func(migrator *migrate.Migrate) error {
		r.Infof("Running migrations down to empty database...")
		return migrator.Down()
	})
}

func (r *GolangMigrateRunner) Goto(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString, version uint) error {
	return r.runMigrationFunction(ctx, driver, connectionString, func(migrator *migrate.Migrate) error {
		r.Infof("Running migrations to go to version %d...", version)
		return migrator.Migrate(version)
	})
}

func (r *GolangMigrateRunner) Force(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString, version uint) error {
	return r.runMigrationFunction(ctx, driver, connectionString, func(migrator *migrate.Migrate) error {
		r.Infof("Running force migration to set database at version %d...", version)
		return migrator.Force(int(version))
	})
}

// runMigrationFunction Sets up a golang-migrate migrator attached to the specified database, and then
// runs the supplied function to perform one or more migrations.
// Note that the golang-migrate library does not take a context, so this is ignored.
func (r *GolangMigrateRunner) runMigrationFunction(
	ctx context.Context,
	driver store.DBDriver,
	connectionString store.DatabaseConnectionString,
	fn func(*migrate.Migrate) error,
) error {
	// Produce migration files targeted to the specific SQL dialect, on an in-memory filesystem
	dialectTemplate, err := GetDialectForDriver(driver)
	if err != nil {
		return err
	}
	inMemoryFS, err := r.ProduceMigrationFiles(dialectTemplate)
	if err != nil {
		return err
	}

	sourceDriver, err := migrate_iofs.New(inMemoryFS, "migrations")
	if err != nil {
		return err
	}

	// Make a separate database for the golang-migrate runner, which will close the database
	sqlxDB, err := sqlx.Open(string(driver), string(connectionString))
	if err != nil {
		return fmt.Errorf("error opening %s database for migration: %w", driver, err)
	}
	databaseDriver, err := r.getMigrationDriverForExistingDatabase(sqlxDB)
	if err != nil {
		sqlxDB.Close()
		return err
	}

	// Set up 'Migrate' instance to source migrations from iofs and apply to the database
	migrator, err := migrate.NewWithInstance("iofs", sourceDriver, "sqlite", databaseDriver)
	if err != nil {
		sqlxDB.Close()
		return err
	}
	defer migrator.Close() // this will close the database so from here on this doesn't need to be explicitly done

	err = fn(migrator)
	if err != nil {
		if err == migrate.ErrNoChange {
			r.Infof("No change needed from migrations")
			err = nil
		} else {
			return err
		}
	} else {
		r.Infof("Migration completed successfully.")
	}

	return nil
}

// getMigrationDriverForExistingDatabase will set up a golang-migrate database driver for the supplied existing
// database, opening a new connection to the database if required.
func (r *GolangMigrateRunner) getMigrationDriverForExistingDatabase(db *sqlx.DB) (driver migrate_database.Driver, err error) {
	switch db.DriverName() {
	case store.Sqlite.String():
		migrateConfig := &migrate_sqlite3.Config{
			DatabaseName:    "sqlite", // database name is ignored for sqlite
			MigrationsTable: "",       // DefaultMigrationsTable will be used if no table supplied
			NoTxWrap:        false,    // wrap statements in a transaction
		}
		driver, err = migrate_sqlite3.WithInstance(db.DB, migrateConfig)
		if err != nil {
			return nil, fmt.Errorf("error creating migration database driver instance for Sqlite: %w", err)
		}
		return driver, err
	case store.Postgres.String():
		migrateConfig := &migrate_postgres.Config{
			DatabaseName:          "", // database name will be filled out from existing database
			MigrationsTable:       "", // DefaultMigrationsTable will be used if no table supplied
			MigrationsTableQuoted: false,
			StatementTimeout:      5 * time.Second, // use a sensible timeout
			MultiStatementEnabled: true,            // we often have multiple statements in one migration
			MultiStatementMaxSize: migrate_postgres.DefaultMultiStatementMaxSize,
		}
		driver, err = migrate_postgres.WithInstance(db.DB, migrateConfig)
		if err != nil {
			return nil, fmt.Errorf("error creating migration database driver instance for Postgres: %w", err)
		}
		return driver, err
	}

	return nil, fmt.Errorf("error unsupported migration database driver: %s", db.DriverName())
}

// ProduceMigrationFiles produces a set of migration files for the specified set of migrations, customised for
// the specified SQL dialect using templating, and suitable for golang-migrate to process.
// The files are written to an in-memory filesystem which is returned and can be accessed via fs.io
func (r *GolangMigrateRunner) ProduceMigrationFiles(dialectTemplate *DialectTemplate) (*memfs.FS, error) {
	inMemoryFS := memfs.New()

	// Make migrations directory
	err := inMemoryFS.MkdirAll("migrations", 0777)
	if err != nil {
		return nil, err
	}

	r.Debugf("Templating migrations")
	for _, migration := range r.migrationData {
		err = r.writeMigrationFile(inMemoryFS, dialectTemplate, migration.SequenceNumber, migration.Name, "up", migration.UpSQL)
		if err != nil {
			return nil, err
		}
		err = r.writeMigrationFile(inMemoryFS, dialectTemplate, migration.SequenceNumber, migration.Name, "down", migration.DownSQL)
		if err != nil {
			return nil, err
		}
	}
	return inMemoryFS, nil
}

func (r *GolangMigrateRunner) writeMigrationFile(
	inMemoryFS *memfs.FS,
	dialectTemplate *DialectTemplate,
	sequenceNumber int64,
	migrationName string,
	upOrDown string,
	sql string,
) error {
	// File name format for a migration is '{version}_{title}.{up-or-down}.{extension}'
	migrationPath := fmt.Sprintf("migrations/%06d_%s.%s.sql", sequenceNumber, migrationName, upOrDown)
	r.Debugf("Templating migration: %s", migrationPath)
	migrationTemplate, err := template.New(migrationName).Parse(sql)
	if err != nil {
		return fmt.Errorf("error parsing migration '(%s)' template: %w", migrationPath, err)
	}
	var migrationBuffer bytes.Buffer
	err = migrationTemplate.Execute(&migrationBuffer, dialectTemplate)
	if err != nil {
		return fmt.Errorf("error applying migration '%s' template: %w", migrationPath, err)
	}

	err = inMemoryFS.WriteFile(migrationPath, migrationBuffer.Bytes(), 0755)
	if err != nil {
		return fmt.Errorf("error writing migration '%s' (%s) to in-memory filesystem: %w", migrationPath, upOrDown, err)
	}
	return nil
}
