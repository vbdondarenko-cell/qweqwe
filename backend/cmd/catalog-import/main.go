package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/catalog"
	"github.com/vbdondarenko-cell/qweqwe/backend/internal/postgres"
)

func main() {
	filePath := flag.String("file", "", "strict JSON catalog file")
	dryRun := flag.Bool("dry-run", false, "validate only; do not connect to PostgreSQL")
	flag.Parse()
	if *filePath == "" {
		log.Fatal("--file is required")
	}
	file, err := os.Open(*filePath)
	if err != nil {
		log.Fatalf("open catalog: %v", err)
	}
	defer file.Close()
	doc, err := catalog.Decode(file)
	if err != nil {
		log.Fatalf("validate catalog: %v", err)
	}
	if *dryRun {
		fmt.Printf("catalog valid: localities=%d places=%d\n", len(doc.Localities), len(doc.Places))
		return
	}
	dsn := os.Getenv("LINKUP_CATALOG_DATABASE_URL")
	if dsn == "" {
		log.Fatal("LINKUP_CATALOG_DATABASE_URL is required for write import")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		log.Fatalf("database configuration: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		log.Fatalf("database unavailable: %v", err)
	}
	store, err := postgres.NewCatalogImportStore(pool)
	if err != nil {
		log.Fatalf("catalog importer unavailable: %v", err)
	}
	result, err := store.Import(ctx, doc)
	if err != nil {
		if errors.Is(err, catalog.ErrInvalidCatalog) {
			log.Fatal("catalog references are invalid")
		}
		log.Fatalf("catalog import failed: %v", err)
	}
	fmt.Printf("catalog imported: localities=%d places=%d\n", result.LocalitiesUpserted, result.PlacesUpserted)
}
