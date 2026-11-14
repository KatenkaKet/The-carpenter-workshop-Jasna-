package database

import (
	"context"
	"embed"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
)

//go:embed migrations/*.sql
var migrations embed.FS

func Open(ctx context.Context, url string) (*pgxpool.Pool, error) {
	c, e := pgxpool.ParseConfig(url)
	if e != nil {
		return nil, e
	}
	c.ConnConfig.RuntimeParams["search_path"] = "jasna,public"
	c.ConnConfig.RuntimeParams["timezone"] = "UTC"
	c.MaxConns = 12
	db, e := pgxpool.NewWithConfig(ctx, c)
	if e != nil {
		return nil, e
	}
	if e = db.Ping(ctx); e != nil {
		db.Close()
		return nil, e
	}
	return db, nil
}

func Migrate(ctx context.Context, db *pgxpool.Pool) error {
	tx, e := db.Begin(ctx)
	if e != nil {
		return e
	}
	defer tx.Rollback(ctx)
	if _, e = tx.Exec(ctx, "SELECT pg_advisory_xact_lock(7341021)"); e != nil {
		return e
	}
	if _, e = tx.Exec(ctx, `CREATE SCHEMA IF NOT EXISTS jasna; CREATE TABLE IF NOT EXISTS jasna.schema_migrations(version integer PRIMARY KEY, applied_at timestamptz NOT NULL DEFAULT now())`); e != nil {
		return e
	}
	var exists bool
	if e = tx.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM jasna.schema_migrations WHERE version=1)").Scan(&exists); e != nil {
		return e
	}
	if !exists {
		sql, _ := migrations.ReadFile("migrations/001_init.sql")
		if _, e = tx.Exec(ctx, string(sql)); e != nil {
			return fmt.Errorf("migration 1: %w", e)
		}
		if _, e = tx.Exec(ctx, "INSERT INTO jasna.schema_migrations(version) VALUES(1)"); e != nil {
			return e
		}
	}
	return tx.Commit(ctx)
}
