package migrations_test

import (
	"context"
	"io/fs"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/migrations"
	"github.com/buildbeaver/buildbeaver/server/store/store_test"
)

const inMemorySqliteConnectionString = store.DatabaseConnectionString("file::memory:?cache=shared&_foreign_keys=1&parseTime=true")

var migrationTestData = migrations.MigrationSet{
	{
		SequenceNumber: 1,
		Name:           "create_test_people",
		UpSQL: `CREATE TABLE IF NOT EXISTS test_people
				(
					person_id text NOT NULL PRIMARY KEY,
					person_name text NOT NULL,
					person_created_at timestamp without time zone NOT NULL,
					person_deleted_at timestamp without time zone,
					person_picture {{ .Binary}}
				);
				CREATE UNIQUE INDEX IF NOT EXISTS test_people_name_unique_index ON test_people(person_name)
				WHERE person_deleted_at IS NULL;
				CREATE UNIQUE INDEX test_people_created_at_id_desc_unique_index ON test_people(
					person_created_at DESC,
					person_id DESC);`,
		DownSQL: `DROP TABLE test_people;`,
	},
	{
		SequenceNumber: 2,
		Name:           "create_test_parents",
		UpSQL: `CREATE TABLE test_parent_relationships
				(
				   parent_relationship_id {{ .IntegerPrimaryKey}},
				   parent_relationship_parent_id text NOT NULL REFERENCES test_people (person_id) ON UPDATE NO ACTION ON DELETE CASCADE,
				   parent_relationship_child_id text NOT NULL REFERENCES test_people (person_id) ON UPDATE NO ACTION ON DELETE CASCADE
				);`,
		DownSQL: `DROP TABLE test_parent_relationships;`,
	},
	{
		SequenceNumber: 3,
		Name:           "alter_test_parents",
		UpSQL:          `ALTER TABLE test_parent_relationships ADD person_address text;`,
		DownSQL:        `ALTER TABLE test_parent_relationships DROP COLUMN person_address;`,
	},
}

func TestMigrations(t *testing.T) {
	logRegistry, err := logger.NewLogRegistry("")
	require.NoError(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	// Test migrations using an in-memory sqlite database
	t.Run("sqlite-in-memory", testMigrationsForDB(store.Sqlite, inMemorySqliteConnectionString, false, logFactory))

	// Set up our default test database, configured via environment variables (could be any database)
	t.Log("Setting up test database")
	database, cleanup, err := store_test.ConnectAndOptionallyMigrate(false, logFactory)
	require.NoError(t, err)
	defer cleanup()
	t.Run("default-test-database", testMigrationsForDB(database.Driver, database.ConnectionString, true, logFactory))
}

// testMigrations runs various migration tests using the migrationTestData against the database with the
// specified driver and connection string.
func testMigrationsForDB(
	driver store.DBDriver,
	connectionString store.DatabaseConnectionString,
	expectFailAfterForce bool,
	logFactory logger.LogFactory,
) func(t *testing.T) {
	return func(t *testing.T) {
		ctx := context.Background()
		migrationRunner := migrations.NewGolangMigrateRunner(migrationTestData, logFactory)

		// Run the first Up migration
		t.Log("Running Up migration 1")
		err := migrationRunner.Up(ctx, driver, connectionString)
		require.NoError(t, err)

		// Repeat the migrations; this will be a no-op
		t.Log("Running Up migration2 ")
		err = migrationRunner.Up(ctx, driver, connectionString)
		require.NoError(t, err)

		// Reverse all migrations
		err = migrationRunner.Down(ctx, driver, connectionString)
		t.Log("Running Down migration 1")
		require.NoError(t, err)

		// Run all migrations again
		t.Log("Running Up migration 3")
		err = migrationRunner.Up(ctx, driver, connectionString)
		require.NoError(t, err)

		// Go back to migration 2
		t.Log("Running Goto 2 migration")
		err = migrationRunner.Goto(ctx, driver, connectionString, 2)
		require.NoError(t, err)

		// Go back to migration 1
		t.Log("Running Goto 1 migration")
		err = migrationRunner.Goto(ctx, driver, connectionString, 1)
		require.NoError(t, err)

		// Force migrations to 3; the database is really only at 1 but this should succeed
		t.Log("Running Force 3 migration")
		err = migrationRunner.Force(ctx, driver, connectionString, 3)
		require.NoError(t, err)

		// Try to run down migration; this should fail since we never actually ran up migrations 2 and 3.
		// Note that an 'up' migration here would appear to succeed but would not actually run migrations 2 and 3.
		// Note also that this down migration succeeds for an in-memory sqlite database which seems to be more
		// permissive
		t.Log("Running Down migration 2")
		err = migrationRunner.Down(ctx, driver, connectionString)
		if expectFailAfterForce {
			require.Error(t, err)
		} else {
			require.NoError(t, err)
		}

		// Force migrations back to 1; this will 'fix' the database
		t.Log("Running Force 1 migration")
		err = migrationRunner.Force(ctx, driver, connectionString, 1)
		require.NoError(t, err)

		// Try to run down migration again; this should now succeed
		t.Log("Running Down migration 2")
		err = migrationRunner.Down(ctx, driver, connectionString)
		require.NoError(t, err)

		// Run all migrations again
		t.Log("Running Up migration 4")
		err = migrationRunner.Up(ctx, driver, connectionString)
		require.NoError(t, err)
	}
}

func TestMigrationTemplating(t *testing.T) {
	t.Run("Sqlite", testMigrationTemplating(migrations.NewSqliteDialectTemplate()))
	t.Run("Postgres", testMigrationTemplating(migrations.NewPostgresDialectTemplate()))
}

func testMigrationTemplating(dialectTemplate *migrations.DialectTemplate) func(t *testing.T) {
	return func(t *testing.T) {
		logRegistry, err := logger.NewLogRegistry("")
		require.NoError(t, err)
		logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

		migrationRunner := migrations.NewBBGolangMigrateRunner(logFactory)

		// Produce migration files for postgres
		inMemoryFS, err := migrationRunner.ProduceMigrationFiles(dialectTemplate)
		require.NoError(t, err)

		// Walk the directory tree and output filenames
		err = fs.WalkDir(inMemoryFS, ".", func(path string, d fs.DirEntry, err error) error {
			t.Logf("Produced migration file: %s", path)
			return nil
		})
		require.NoError(t, err)
	}
}

// TestServerMigrations will test the migrations for the BuildBeaver server, both up and down, with
// a database as would be set up by default for our tests. The actual database server used will be configured using
// environment variables, and a new test database will be created for those database servers that support it.
func TestServerMigrations(t *testing.T) {
	// Set up logging
	logRegistry, err := logger.NewLogRegistry("")
	require.NoError(t, err)
	logFactory := logger.MakeLogrusLogFactoryStdOut(logRegistry)

	ctx := context.Background()

	// Set up our default test database, configured via environment variables (could be any database)
	// Test asking ConnectAndOptionallyMigrate() to run the 'up' migrations
	t.Log("Setting up test database (including Up migration 1)")
	database, cleanup, err := store_test.ConnectAndOptionallyMigrate(true, logFactory)
	require.NoError(t, err)
	defer cleanup()

	migrationRunner := migrations.NewBBGolangMigrateRunner(logFactory)

	// Repeat the migrations; this will be a no-op
	t.Log("Running Up migration 2")
	err = migrationRunner.Up(ctx, database.Driver, database.ConnectionString)
	require.NoError(t, err)

	// Reverse all migrations
	err = migrationRunner.Down(ctx, database.Driver, database.ConnectionString)
	t.Log("Running Down migration 1")
	require.NoError(t, err)

	// Run all migrations again
	t.Log("Running Up migration 3")
	err = migrationRunner.Up(ctx, database.Driver, database.ConnectionString)
	require.NoError(t, err)
}
