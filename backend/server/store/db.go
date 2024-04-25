package store

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/doug-martin/goqu/v9"
	_ "github.com/doug-martin/goqu/v9/dialect/postgres"
	_ "github.com/doug-martin/goqu/v9/dialect/sqlite3"
	"github.com/jmoiron/sqlx"
	_ "github.com/mattn/go-sqlite3"
	"github.com/pkg/errors"
)

type DatabaseConfig struct {
	ConnectionString   DatabaseConnectionString
	Driver             DBDriver
	MaxIdleConnections int
	MaxOpenConnections int
}

type DBDriver string

func (d DBDriver) String() string {
	return string(d)
}

type DatabaseConnectionString string

func (d DatabaseConnectionString) String() string {
	return string(d)
}

const (
	Sqlite                            DBDriver = "sqlite3"
	Postgres                          DBDriver = "postgres"
	DefaultDatabaseMaxIdleConnections          = 2
	DefaultDatabaseMaxOpenConnections          = 4
)

type DBMigrator interface {
	Up(db *DB) error
	Down(db *DB) error
}

type DB struct {
	*sqlx.DB
	Driver           DBDriver
	ConnectionString DatabaseConnectionString
	lock             sync.RWMutex
}

type Tx struct {
	tx *sqlx.Tx
}

// A Scanner represents an object that can be scanned for values.
type Scanner interface {
	Scan(model ...interface{}) error
}

// Binder interface defines database field bindings.
type Binder interface {
	BindNamed(query string, arg interface{}) (string, []interface{}, error)
}

// Queryer interface defines a set of methods for querying the database.
type Queryer interface {
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

// Execer interface defines a set of methods for executing
// read and write commands against the database.
type Execer interface {
	Queryer
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

// MigrationRunner interface defines a set of methods for applying database migrations.
type MigrationRunner interface {
	// Up migrates the given database up to the latest version.
	Up(ctx context.Context, driver DBDriver, connectionString DatabaseConnectionString) error
	// Down migrates the given database down to empty.
	Down(ctx context.Context, driver DBDriver, connectionString DatabaseConnectionString) error
	// Goto migrates the given database to the specified version.
	Goto(ctx context.Context, driver DBDriver, connectionString DatabaseConnectionString, version uint) error
	// Force marks the database as clean and already migrated to the specified version.
	Force(ctx context.Context, driver DBDriver, connectionString DatabaseConnectionString, version uint) error
}

// NewDatabase performs any database specific init required before returning a new database connection
// pool using the specified DatabaseConfig, as well as a cleanup function to call to close the database again.
// If a MigrationRunner is supplied then an 'Up' migration will be performed to ensure the database schema
// is up to the latest version.
func NewDatabase(
	ctx context.Context,
	config DatabaseConfig,
	migrationRunner MigrationRunner,
) (*DB, func(), error) {
	switch config.Driver {
	case Sqlite:
		err := SQLiteConnectionInit(string(config.ConnectionString))
		if err != nil {
			return nil, nil, err
		}
		break
	case Postgres:
		// TODO: Any init required for postgres
		break
	default:
		return nil, nil, fmt.Errorf("unknown database Driver %s", config.Driver)
	}

	sqlxDB, err := sqlx.Open(string(config.Driver), string(config.ConnectionString))
	if err != nil {
		return nil, nil, fmt.Errorf("error opening %s database: %w", config.Driver, err)
	}

	err = sqlxDB.PingContext(ctx)
	if err != nil {
		sqlxDB.Close()
		return nil, nil, fmt.Errorf("error pinging %s database: %w", config.Driver, err)
	}

	// Apply database migrations to ensure schema is up to the latest version
	if migrationRunner != nil {
		err := migrationRunner.Up(ctx, config.Driver, config.ConnectionString)
		if err != nil {
			sqlxDB.Close()
			return nil, nil, fmt.Errorf("error running %s database migrations: %w", config.Driver, err)
		}
	}

	db := &DB{
		DB:               sqlxDB,
		Driver:           config.Driver,
		ConnectionString: config.ConnectionString,
	}

	// Apply idle and open connection configurations
	db.DB.SetMaxIdleConns(config.MaxIdleConnections)
	db.DB.SetMaxOpenConns(config.MaxOpenConnections)
	cleanup := func() {
		db.Close()
	}
	return db, cleanup, nil
}

// SQLiteConnectionInit performs any initialization required for SQLite.
// Currently, this is to create the local db file if a file based connection string is used.
func SQLiteConnectionInit(connectionString string) error {
	// https://github.com/mattn/go-sqlite3/issues/677
	// TL;DR: In-memory connection strings contain both a :memory: and a file: directive.
	if strings.Contains(connectionString, ":memory:") {
		return nil
	}

	const sqliteFileKeyword = "file:"
	var databaseFilePath string
	s := strings.Index(connectionString, sqliteFileKeyword)
	if s == -1 {
		return nil
	}
	s += len(sqliteFileKeyword)
	e := strings.Index(connectionString[s:], "?")
	if e == -1 {
		databaseFilePath = connectionString[s:]
	} else {
		databaseFilePath = connectionString[s : s+e]
	}

	dir := filepath.Dir(databaseFilePath)
	err := os.MkdirAll(dir, 0755)
	if err != nil {
		return fmt.Errorf("error ensuring database directory %q exists: %w", dir, err)
	}

	file, err := os.OpenFile(string(databaseFilePath), os.O_RDONLY|os.O_CREATE, 0660)
	if err != nil {
		return fmt.Errorf("error opening or creating database file %q: %w", databaseFilePath, err)
	}

	err = file.Close()
	if err != nil {
		return fmt.Errorf("error closing database file: %w", err)
	}

	return nil
}

// WithTx runs fn inside a database transaction. If fn returns an error the
// transaction will be rolled back and aborted. If fn returns nil, that transaction
// will be committed. If ctx is cancelled or deadlines before the transaction is
// committed the transaction will be rolled back and aborted.
func (d *DB) WithTx(ctx context.Context, txOrNil *Tx, fn func(tx *Tx) error) error {

	if txOrNil != nil {
		return fn(txOrNil)
	}

	if d.Driver == Sqlite {
		d.lock.Lock()
		defer d.lock.Unlock()
	}

	tx, err := d.DB.BeginTxx(ctx, nil)
	if err != nil {
		return errors.Wrap(err, "error beginning database transaction")
	}

	err = fn(&Tx{tx})
	if err != nil {
		originalErr := err
		err = tx.Rollback()
		if err != nil {
			return errors.Wrapf(err, "error rolling back database transaction: %s", originalErr)
		}
		return originalErr
	}

	err = tx.Commit()
	if err != nil {
		return errors.Wrap(err, "error committing database transaction")
	}

	return nil
}

// Write prepares the database for writing and calls fn() with the Execer
// to use to write to the database. If Tx is supplied, Execer will be bound
// to the transaction, otherwise a new implicit transaction will be started.
func (d *DB) Write(txOrNil *Tx, fn func(Execer, Binder) error) error {
	if txOrNil == nil {
		if d.Driver == Sqlite {
			d.lock.Lock()
			defer d.lock.Unlock()
		}
		return fn(d.DB, d.DB)
	}
	return fn(txOrNil.tx, txOrNil.tx)
}

// Read prepares the database for reading and calls fn() with the Queryer
// to use to read from the database. If Tx is supplied, Queryer will be bound
// to the transaction, otherwise a new implicit transaction will be started.
func (d *DB) Read(txOrNil *Tx, fn func(Queryer, Binder) error) error {
	if txOrNil == nil {
		if d.Driver == Sqlite {
			d.lock.RLock()
			defer d.lock.RUnlock()
		}
		return fn(d.DB, d.DB)
	}
	return fn(txOrNil.tx, txOrNil.tx)
}

// Close the connection to the database. The DB object must not be used
// after a call to Close.
func (d *DB) Close() error {
	return d.DB.Close()
}

func (d *DB) Write2(txOrNil *Tx, fn func(Writer) error) error {
	if txOrNil == nil {
		if d.Driver == Sqlite {
			d.lock.Lock()
			defer d.lock.Unlock()
		}
		return fn(goqu.New(d.DriverName(), d.DB))
	}
	return fn(goqu.NewTx(d.DriverName(), txOrNil.tx))
}

func (d *DB) Read2(txOrNil *Tx, fn func(Reader) error) error {
	if txOrNil == nil {
		if d.Driver == Sqlite {
			d.lock.RLock()
			defer d.lock.RUnlock()
		}
		return fn(goqu.New(d.DriverName(), d.DB))
	}
	return fn(goqu.NewTx(d.DriverName(), txOrNil.tx))
}

// SupportsRowLevelLocking returns true if the current database supports the 'SELECT ... FOR UPDATE'
// syntax to lock table rows, or false if these locks are not required (e.g. sqlite)
func (d *DB) SupportsRowLevelLocking() bool {
	if d.Driver == Sqlite {
		return false
	}
	// Assume other databases support this
	return true
}

type Writer interface {
	Reader
	Update(table interface{}) *goqu.UpdateDataset
	Insert(table interface{}) *goqu.InsertDataset
	Delete(table interface{}) *goqu.DeleteDataset
	Truncate(table ...interface{}) *goqu.TruncateDataset
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
}

type Reader interface {
	From(from ...interface{}) *goqu.SelectDataset
	Select(cols ...interface{}) *goqu.SelectDataset
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	ScanStructsContext(ctx context.Context, i interface{}, query string, args ...interface{}) error
	ScanStructContext(ctx context.Context, i interface{}, query string, args ...interface{}) (bool, error)
	ScanValsContext(ctx context.Context, i interface{}, query string, args ...interface{}) error
	ScanValContext(ctx context.Context, i interface{}, query string, args ...interface{}) (bool, error)
}
