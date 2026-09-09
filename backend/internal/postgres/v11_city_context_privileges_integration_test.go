package postgres

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/migrate"
)

func TestV11CityContextRuntimePrivileges(t *testing.T) {
	dsn := os.Getenv("LINKUP_TEST_DATABASE_URL")
	if dsn == "" || os.Getenv("LINKUP_TEST_DATABASE_DESTRUCTIVE") != "1" {
		t.Skip("disposable PostgreSQL requires explicit destructive opt-in")
	}
	ctx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()
	if err := migrate.Apply(ctx, pool, migrationDir(t)); err != nil {
		t.Fatal(err)
	}

	var roleExists bool
	if err := pool.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM pg_roles WHERE rolname='linkup_api')`).Scan(&roleExists); err != nil {
		t.Fatal(err)
	}
	if !roleExists {
		t.Skip("linkup_api role is not present in disposable PostgreSQL")
	}

	var localitySelect, localityWrite, placeSelect, placeWrite, lockSelect, lockInsert, lockUpdate, lockDelete bool
	err = pool.QueryRow(ctx, `SELECT
		has_table_privilege('linkup_api','public.localities','SELECT'),
		has_table_privilege('linkup_api','public.localities','INSERT') OR has_table_privilege('linkup_api','public.localities','UPDATE') OR has_table_privilege('linkup_api','public.localities','DELETE'),
		has_table_privilege('linkup_api','public.canonical_places','SELECT'),
		has_table_privilege('linkup_api','public.canonical_places','INSERT') OR has_table_privilege('linkup_api','public.canonical_places','UPDATE') OR has_table_privilege('linkup_api','public.canonical_places','DELETE'),
		has_table_privilege('linkup_api','public.city_context_locks','SELECT'),
		has_table_privilege('linkup_api','public.city_context_locks','INSERT'),
		has_table_privilege('linkup_api','public.city_context_locks','UPDATE'),
		has_table_privilege('linkup_api','public.city_context_locks','DELETE')`).
		Scan(&localitySelect, &localityWrite, &placeSelect, &placeWrite, &lockSelect, &lockInsert, &lockUpdate, &lockDelete)
	if err != nil {
		t.Fatal(err)
	}
	if !localitySelect || localityWrite || !placeSelect || placeWrite || !lockSelect || !lockInsert || !lockUpdate || lockDelete {
		t.Fatalf("unexpected linkup_api City Context privileges localitySelect=%v localityWrite=%v placeSelect=%v placeWrite=%v lockSelect=%v lockInsert=%v lockUpdate=%v lockDelete=%v",
			localitySelect, localityWrite, placeSelect, placeWrite, lockSelect, lockInsert, lockUpdate, lockDelete)
	}
}
