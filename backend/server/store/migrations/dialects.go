package migrations

import (
	"fmt"

	"github.com/buildbeaver/buildbeaver/server/store"
)

func NewPostgresDialectTemplate() *DialectTemplate {
	return &DialectTemplate{
		Binary:            "BYTEA",
		IntegerPrimaryKey: "SERIAL PRIMARY KEY",
	}
}

func NewSqliteDialectTemplate() *DialectTemplate {
	return &DialectTemplate{
		Binary:            "BLOB",
		IntegerPrimaryKey: "integer NOT NULL PRIMARY KEY AUTOINCREMENT",
	}
}

func GetDialectForDriver(driver store.DBDriver) (*DialectTemplate, error) {
	switch driver {
	case store.Sqlite:
		return NewSqliteDialectTemplate(), nil
	case store.Postgres:
		return NewPostgresDialectTemplate(), nil
	}

	return nil, fmt.Errorf("error unsupported database driver: %s", driver)
}
