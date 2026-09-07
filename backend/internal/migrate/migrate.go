package migrate

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// A fixed transaction-scoped advisory lock serializes LinkUp migration ledger
// decisions across concurrent API processes. pg_advisory_xact_lock is released
// automatically on commit/rollback, including context cancellation.
const migrationAdvisoryLock int64 = 0x4c696e6b55703130 // "LinkUp10"

type migration struct {
	Name     string
	SQL      string
	Checksum [sha256.Size]byte
}

func Apply(ctx context.Context, pool *pgxpool.Pool, dir string) error {
	if pool == nil {
		return errors.New("nil database pool")
	}
	if _, err := pool.Exec(ctx, `CREATE TABLE IF NOT EXISTS linkup_schema_migrations (name text PRIMARY KEY, checksum bytea NOT NULL, applied_at timestamptz NOT NULL DEFAULT now())`); err != nil {
		return fmt.Errorf("ensure migration ledger: %w", err)
	}
	items, err := discover(dir)
	if err != nil {
		return err
	}
	for _, m := range items {
		if err := applyOne(ctx, pool, m); err != nil {
			return err
		}
	}
	return nil
}

func applyOne(ctx context.Context, pool *pgxpool.Pool, m migration) error {
	tx, err := pool.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return fmt.Errorf("begin migration %s: %w", m.Name, err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `SELECT pg_advisory_xact_lock($1)`, migrationAdvisoryLock); err != nil {
		return fmt.Errorf("lock migration %s: %w", m.Name, err)
	}

	var existing []byte
	err = tx.QueryRow(ctx, `SELECT checksum FROM linkup_schema_migrations WHERE name=$1`, m.Name).Scan(&existing)
	if err == nil {
		if len(existing) != len(m.Checksum) || !equal(existing, m.Checksum[:]) {
			return fmt.Errorf("migration %s checksum drift detected", m.Name)
		}
		return tx.Commit(ctx)
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("read migration %s: %w", m.Name, err)
	}

	executable, err := executionSQL(m.SQL)
	if err != nil {
		return fmt.Errorf("prepare migration %s: %w", m.Name, err)
	}
	if _, err = tx.Exec(ctx, executable); err != nil {
		return fmt.Errorf("apply migration %s: %w", m.Name, err)
	}
	if _, err = tx.Exec(ctx, `INSERT INTO linkup_schema_migrations (name,checksum) VALUES ($1,$2)`, m.Name, m.Checksum[:]); err != nil {
		return fmt.Errorf("record migration %s: %w", m.Name, err)
	}
	if err = tx.Commit(ctx); err != nil {
		return fmt.Errorf("commit migration %s: %w", m.Name, err)
	}
	return nil
}

// executionSQL keeps migration checksums immutable while making transaction
// ownership explicit. The runner always owns the PostgreSQL transaction. A
// previously committed migration may still contain one legacy full-file
// `BEGIN; ... COMMIT;` wrapper; that wrapper is removed only for execution.
// One-sided wrappers are rejected so a migration cannot silently commit or
// escape the runner's checksum-ledger transaction.
func executionSQL(raw string) (string, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return "", errors.New("empty migration SQL")
	}

	const begin = "BEGIN;"
	const commit = "COMMIT;"
	hasBegin := len(s) >= len(begin) && strings.EqualFold(s[:len(begin)], begin)
	hasCommit := len(s) >= len(commit) && strings.EqualFold(s[len(s)-len(commit):], commit)

	if hasBegin != hasCommit {
		return "", errors.New("unbalanced migration transaction wrapper")
	}
	if !hasBegin {
		return s, nil
	}

	body := strings.TrimSpace(s[len(begin) : len(s)-len(commit)])
	if body == "" {
		return "", errors.New("empty migration transaction body")
	}
	return body, nil
}

func discover(dir string) ([]migration, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("read migration directory: %w", err)
	}
	items := make([]migration, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		b, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return nil, fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		items = append(items, migration{Name: entry.Name(), SQL: string(b), Checksum: sha256.Sum256(b)})
	}
	sort.Slice(items, func(i, j int) bool { return items[i].Name < items[j].Name })
	return items, nil
}

func equal(a, b []byte) bool {
	if len(a) != len(b) {
		return false
	}
	var v byte
	for i := range a {
		v |= a[i] ^ b[i]
	}
	return v == 0
}
