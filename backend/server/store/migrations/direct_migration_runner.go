package migrations

import (
	"bytes"
	"context"
	"fmt"
	"text/template"

	"github.com/jmoiron/sqlx"

	"github.com/buildbeaver/buildbeaver/common/logger"
	"github.com/buildbeaver/buildbeaver/server/store"
)

// DirectMigrationRunner runs 'up' migrations directly, always running all migrations each time.
// This is now deprecated; use GolangMigrateRunner instead.
type DirectMigrationRunner struct {
	logger.Log
}

func NewDirectMigrationRunner(
	logFactory logger.LogFactory,
) *DirectMigrationRunner {
	return &DirectMigrationRunner{
		Log: logFactory("DirectMigrationRunner"),
	}
}

func (r *DirectMigrationRunner) Up(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString) error {
	dialectTemplate, err := GetDialectForDriver(driver)
	if err != nil {
		return err
	}

	sqlxDB, err := sqlx.Open(string(driver), string(connectionString))
	if err != nil {
		return fmt.Errorf("error opening %s database for migration: %w", driver, err)
	}
	defer sqlxDB.Close()
	db := &store.DB{
		DB:               sqlxDB,
		Driver:           driver,
		ConnectionString: connectionString,
	}

	// Run all 'up' migrations inside a transaction
	return db.WithTx(ctx, nil, func(tx *store.Tx) error {
		r.Debugf("Running migrations")
		for _, migration := range BuildBeaverServerMigrations {
			r.Debugf("Running migration: %s", migration.Name)
			t, err := template.New(migration.Name).Parse(migration.UpSQL)
			if err != nil {
				return fmt.Errorf("error parsing migration '(%s)' template: %w", migration.Name, err)
			}
			var tpl bytes.Buffer
			err = t.Execute(&tpl, dialectTemplate)
			if err != nil {
				return fmt.Errorf("error applying migration '(%s)' template: %w", migration.Name, err)
			}
			err = db.Write2(tx, func(writer store.Writer) error {
				_, err = writer.ExecContext(ctx, tpl.String())
				return err
			})
			if err != nil {
				return fmt.Errorf("error executing migration '(%s)' data: %w", migration.Name, err)
			}
		}
		return nil
	})
}

func (r *DirectMigrationRunner) Down(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString) error {
	return fmt.Errorf("error: Down not implemented")
}

func (r *DirectMigrationRunner) Goto(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString, version uint) error {
	return fmt.Errorf("error: Goto not implemented")
}

func (r *DirectMigrationRunner) Force(ctx context.Context, driver store.DBDriver, connectionString store.DatabaseConnectionString, version uint) error {
	return fmt.Errorf("error: Force not implemented")
}
