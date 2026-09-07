package postgres

import (
	"context"
	"strings"
	"testing"
)

func TestOpenRedactsMalformedDatabaseURL(t *testing.T) {
	const secret = "supersecret"
	_, err := Open(context.Background(), "postgresql://user:"+secret+"@%zz")
	if err == nil {
		t.Fatal("malformed database URL must fail")
	}
	if strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), "%zz") {
		t.Fatalf("database error leaked raw connection material: %q", err)
	}
	if got := err.Error(); got != "invalid database configuration" {
		t.Fatalf("unexpected safe database error: %q", got)
	}
}
