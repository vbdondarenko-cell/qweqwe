package postgres

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// buildPoolConfig applies connection-level settings shared by every real
// Open() call, kept separate so it can be unit-tested without a live
// database connection.
func buildPoolConfig(databaseURL string) (*pgxpool.Config, error) {
	cfg, err := pgxpool.ParseConfig(databaseURL)
	if err != nil {
		return nil, errors.New("invalid database configuration")
	}
	// The production DATABASE_URL routes through Supabase's PgBouncer pooler
	// in transaction-pooling mode (port 6543), confirmed empirically on the
	// real server (backend IMPLEMENTATION_STATUS.md §58). pgx's default
	// QueryExecModeCacheStatement prepares and caches a named statement per
	// logical connection; transaction pooling can hand two different logical
	// connections the same physical backend mid-session, so a second PREPARE
	// of the identical statement name collides with the first that is still
	// cached on that backend ("prepared statement ... already exists",
	// SQLSTATE 42P05) — this was observed live, repeatedly, from
	// NotificationProjector's polling ticker, which repeats the same query
	// text often enough to make the race likely. SimpleProtocol sends plain
	// SQL text with inlined parameters instead of server-side prepared
	// statements, which is the standard, documented fix for pgx behind a
	// transaction-mode connection pooler (Supabase's own guidance), at the
	// cost of losing statement-level plan caching — an acceptable trade for
	// correctness over a marginal performance gain.
	cfg.ConnConfig.DefaultQueryExecMode = pgx.QueryExecModeSimpleProtocol
	return cfg, nil
}

func Open(ctx context.Context, databaseURL string) (*pgxpool.Pool, error) {
	cfg, err := buildPoolConfig(databaseURL)
	if err != nil {
		return nil, err
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, errors.New("database pool initialization failed")
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, errors.New("database unavailable")
	}
	return pool, nil
}
