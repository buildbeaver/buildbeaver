package store_test

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"net/url"
	"os"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/store"
	"github.com/buildbeaver/buildbeaver/server/store/migrations"
)

func init() {
	rand.Seed(time.Now().UnixNano())
}

const (
	testDBDriverEnvVar         = "TEST_DB_DRIVER"
	testConnectionStringEnvVar = "TEST_CONNECTION_STRING"
)

var letters = []rune("abcdefghijklmnopqrstuvwxyz")

func randSeq(n int) string {
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}

// Connect opens a new test database connection based on environment variables.
// Defaults to in-memory sqlite. Set TEST_DB_DRIVER and TEST_CONNECTION_STRING to select a different database.
// The server migrations will be run against the database.
func Connect(logFactory logger.LogFactory) (*store.DB, func(), error) {
	return ConnectAndOptionallyMigrate(true, logFactory)
}

// ConnectAndOptionallyMigrate opens a new test database connection based on environment variables.
// Defaults to in-memory sqlite. Set TEST_DB_DRIVER and TEST_CONNECTION_STRING to select a different database.
// If runMigrations is true then the server migrations will be run against the database.
func ConnectAndOptionallyMigrate(runMigrations bool, logFactory logger.LogFactory) (*store.DB, func(), error) {
	// Default to SQLite unless we have specified environment variables for the driver and connection string
	var (
		log              = logFactory("TestDB")
		driver           = store.Sqlite
		connectionString = store.DatabaseConnectionString("file::memory:?cache=shared&_foreign_keys=1&parseTime=true")
		cleanupFns       []func()
	)
	val, ok := os.LookupEnv(testDBDriverEnvVar)
	if ok {
		driver = store.DBDriver(val)
		val, ok = os.LookupEnv(testConnectionStringEnvVar)
		if (!ok || val == "") && driver != store.Sqlite {
			return nil, nil, fmt.Errorf("error %s must be set alongside %s when not using sqlite",
				testConnectionStringEnvVar, testDBDriverEnvVar)
		}
		if ok {
			connectionString = store.DatabaseConnectionString(val)
		}
	} else if _, ok = os.LookupEnv(testConnectionStringEnvVar); ok {
		return nil, nil, fmt.Errorf("error %s must be set when using %s",
			testDBDriverEnvVar, testConnectionStringEnvVar)
	}
	if driver == store.Postgres {
		str, cleanup, err := initializeTestDatabase(log, driver, connectionString)
		if err != nil {
			return nil, nil, fmt.Errorf("error initializing test database: %w", err)
		}
		connectionString = str
		cleanupFns = append(cleanupFns, cleanup)
	}

	var migrationRunner store.MigrationRunner = nil
	if runMigrations {
		migrationRunner = migrations.NewBBGolangMigrateRunner(logFactory)
	}

	databaseConfig := store.DatabaseConfig{
		ConnectionString:   connectionString,
		Driver:             driver,
		MaxIdleConnections: store.DefaultDatabaseMaxIdleConnections,
		MaxOpenConnections: store.DefaultDatabaseMaxOpenConnections,
	}

	db, cleanup, err := store.NewDatabase(context.Background(), databaseConfig, migrationRunner)
	if err != nil {
		return nil, nil, fmt.Errorf("error creating database: %w", err)
	}
	cleanupFns = append(cleanupFns, cleanup)

	cleanup = func() {
		log.Info("Running cleanup")
		for i := len(cleanupFns) - 1; i >= 0; i-- {
			cleanupFns[i]()
		}
	}
	return db, cleanup, nil
}

// initializeTestDatabase creates a temporary test database and returns a new connection string to connect to it.
// Call cleanup to drop the database once you're done with it.
// If the original connection string already contains a database in the path returns the original connection string
// and a no-op cleanup function.
func initializeTestDatabase(log logger.Log, driver store.DBDriver, connectionString store.DatabaseConnectionString) (store.DatabaseConnectionString, func(), error) {
	parsed, err := url.Parse(connectionString.String())
	if err != nil {
		return "", nil, fmt.Errorf("error parsing connection string %q: %w", connectionString, err)
	}
	if parsed.Path != "" && parsed.Path != "/" {
		return connectionString, func() {}, nil
	}
	rawDb, err := sql.Open(driver.String(), parsed.String())
	if err != nil {
		return "", nil, fmt.Errorf("error connecting to database: %w", err)
	}
	dbName := fmt.Sprintf("testdb_%s", randSeq(10))
	log.Infof("Creating test database %s", dbName)
	_, err = rawDb.Exec("create database " + dbName)
	if err != nil {
		rawDb.Close()
		return "", nil, fmt.Errorf("error creating database: %w", err)
	}
	cleanup := func() {
		log.Infof("Dropping postgres database %s", dbName)
		_, err := rawDb.Exec("DROP database " + dbName)
		if err != nil {
			log.Fatalf("Failed to drop database: %v", err)
		}
		rawDb.Close()
	}
	parsed.Path = dbName
	return store.DatabaseConnectionString(parsed.String()), cleanup, nil
}
