// Package store holds the app's data in SQLite. Keep one file per concern
// (notes.go, and so on). Every query takes its values as parameters: never
// build SQL from strings.
package store

import (
	"context"
	"database/sql"
	"embed"
	"io/fs"
	"time"

	"github.com/uppertoe/vps-app-kit/sqlitex"
)

//go:embed migrations/*.sql
var migrations embed.FS

// Store wraps the database.
type Store struct {
	db   *sql.DB
	path string
}

// Open opens the database at path, checks it and runs the migrations.
func Open(ctx context.Context, path string) (*Store, error) {
	db, err := sqlitex.Open(path)
	if err != nil {
		return nil, err
	}
	if err := sqlitex.Check(ctx, db); err != nil {
		db.Close()
		return nil, err
	}
	sub, _ := fs.Sub(migrations, "migrations")
	if err := sqlitex.Migrate(ctx, db, sub); err != nil {
		db.Close()
		return nil, err
	}
	return &Store{db: db, path: path}, nil
}

// Close closes the database.
func (s *Store) Close() error { return s.db.Close() }

// Path is the live database's path, for the snapshot.
func (s *Store) Path() string { return s.path }

func now() string { return time.Now().UTC().Format(time.RFC3339Nano) }
