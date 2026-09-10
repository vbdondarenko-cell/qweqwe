package migrate

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestLedgerBridgePinsCanonicalMigrationChecksums(t *testing.T) {
	_, here, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	dir := filepath.Clean(filepath.Join(filepath.Dir(here), "../../../db/migrations"))
	bridgePath := filepath.Join(dir, "000024_migration_ledger_bridge.sql")
	bridgeBytes, err := os.ReadFile(bridgePath)
	if err != nil {
		t.Fatal(err)
	}
	bridge := string(bridgeBytes)

	for number := 1; number <= 23; number++ {
		pattern := filepath.Join(dir, fmt.Sprintf("%06d_*.sql", number))
		matches, err := filepath.Glob(pattern)
		if err != nil {
			t.Fatal(err)
		}
		if len(matches) != 1 {
			t.Fatalf("migration %06d matches=%v", number, matches)
		}
		body, err := os.ReadFile(matches[0])
		if err != nil {
			t.Fatal(err)
		}
		sum := sha256.Sum256(body)
		name := filepath.Base(matches[0])
		needle := fmt.Sprintf("('%s', decode('%x','hex'))", name, sum)
		if !strings.Contains(bridge, needle) {
			t.Fatalf("ledger bridge checksum is stale for %s", name)
		}
	}
	if strings.Contains(bridge, "('000024_migration_ledger_bridge.sql', decode(") {
		t.Fatal("ledger bridge must not pre-record itself; the canonical runner records it after execution")
	}
}
