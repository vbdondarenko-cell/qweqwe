package migrate

import (
	"context"
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

func TestDiscoverSortsAndHashesSQLOnly(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "000002_b.sql"), []byte("SELECT 2;"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "000001_a.sql"), []byte("SELECT 1;"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "notes.txt"), []byte("ignore"), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Name != "000001_a.sql" || items[1].Name != "000002_b.sql" {
		t.Fatalf("unexpected migration order: %#v", items)
	}
	if items[0].Checksum == items[1].Checksum {
		t.Fatal("different SQL produced same checksum")
	}
}

func TestExecutionSQLStripsLegacyWrapperWithoutChangingLedgerChecksum(t *testing.T) {
	raw := "BEGIN;\n\nCREATE TABLE wrapped_probe (id integer);\n\nCOMMIT;\n"
	executable, err := executionSQL(raw)
	if err != nil {
		t.Fatal(err)
	}
	if executable != "CREATE TABLE wrapped_probe (id integer);" {
		t.Fatalf("unexpected executable SQL: %q", executable)
	}

	dir := t.TempDir()
	path := filepath.Join(dir, "000007_wrapped.sql")
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	items, err := discover(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 1 {
		t.Fatalf("expected one migration, got %d", len(items))
	}
	want := sha256.Sum256([]byte(raw))
	if items[0].Checksum != want {
		t.Fatal("execution normalization changed the immutable ledger checksum")
	}
}

func TestExecutionSQLRejectsUnbalancedTransactionWrapper(t *testing.T) {
	for _, raw := range []string{
		"BEGIN;\nSELECT 1;",
		"SELECT 1;\nCOMMIT;",
		"BEGIN;\nCOMMIT;",
		"   ",
	} {
		if _, err := executionSQL(raw); err == nil {
			t.Fatalf("expected rejection for %q", raw)
		}
	}
}

func TestApplySerializesRepeatAndDetectsDrift(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("LINKUP_TEST_DATABASE_URL is required for PostgreSQL migration tests")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	suffix := time.Now().UnixNano()
	table := fmt.Sprintf("linkup_migration_probe_%d", suffix)
	name := fmt.Sprintf("999999_probe_%d.sql", suffix)
	dir := t.TempDir()
	path := filepath.Join(dir, name)
	migrationSQL := fmt.Sprintf("BEGIN;\nCREATE TABLE %s (id integer PRIMARY KEY); INSERT INTO %s(id) VALUES (1); SELECT pg_sleep(0.1);\nCOMMIT;\n", table, table)
	if err := os.WriteFile(path, []byte(migrationSQL), 0600); err != nil {
		t.Fatal(err)
	}
	defer func() {
		_, _ = pool.Exec(context.Background(), "DROP TABLE IF EXISTS "+table)
		_, _ = pool.Exec(context.Background(), `DELETE FROM linkup_schema_migrations WHERE name=$1`, name)
	}()

	start := make(chan struct{})
	errs := make(chan error, 2)
	var wg sync.WaitGroup
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			<-start
			errs <- Apply(ctx, pool, dir)
		}()
	}
	close(start)
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("concurrent Apply failed: %v", err)
		}
	}

	var rows, ledger int
	if err := pool.QueryRow(ctx, "SELECT count(*) FROM "+table).Scan(&rows); err != nil {
		t.Fatal(err)
	}
	if err := pool.QueryRow(ctx, `SELECT count(*) FROM linkup_schema_migrations WHERE name=$1`, name).Scan(&ledger); err != nil {
		t.Fatal(err)
	}
	if rows != 1 || ledger != 1 {
		t.Fatalf("rows=%d ledger=%d; migration effect must occur exactly once", rows, ledger)
	}

	if err := Apply(ctx, pool, dir); err != nil {
		t.Fatalf("repeat Apply must be a no-op: %v", err)
	}
	if err := os.WriteFile(path, []byte(migrationSQL+"\n-- drift"), 0600); err != nil {
		t.Fatal(err)
	}
	if err := Apply(ctx, pool, dir); err == nil || !strings.Contains(err.Error(), "checksum drift detected") {
		t.Fatalf("expected checksum drift rejection, got %v", err)
	}
}
