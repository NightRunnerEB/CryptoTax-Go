package repository

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	db "github.com/NightRunner/CryptoTax-Go/services/aggregation-svc/db/sqlc"
)

func setupIntegrationStore(t *testing.T) db.Store {
	t.Helper()

	dsn := strings.TrimSpace(os.Getenv("AGGREGATION_SVC_TEST_DATABASE_URL"))
	if dsn == "" {
		t.Skip("AGGREGATION_SVC_TEST_DATABASE_URL is not set; skipping integration test")
	}

	ctx := context.Background()

	adminCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse admin dsn: %v", err)
	}
	adminPool, err := pgxpool.NewWithConfig(ctx, adminCfg)
	if err != nil {
		t.Fatalf("create admin pool: %v", err)
	}
	t.Cleanup(adminPool.Close)

	schema := "aggregation_it_" + strings.ReplaceAll(uuid.NewString(), "-", "")
	if _, err := adminPool.Exec(ctx, fmt.Sprintf("CREATE SCHEMA %s", schema)); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = adminPool.Exec(context.Background(), fmt.Sprintf("DROP SCHEMA IF EXISTS %s CASCADE", schema))
	})

	testCfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		t.Fatalf("parse test dsn: %v", err)
	}
	if testCfg.ConnConfig.RuntimeParams == nil {
		testCfg.ConnConfig.RuntimeParams = make(map[string]string)
	}
	testCfg.ConnConfig.RuntimeParams["search_path"] = schema

	testPool, err := pgxpool.NewWithConfig(ctx, testCfg)
	if err != nil {
		t.Fatalf("create test pool: %v", err)
	}
	t.Cleanup(testPool.Close)

	migrationsDir := filepath.Join("../../../db/migrations")
	files := []string{
		"001_create_aggregation_import_state.up.sql",
		"002_create_aggregated_transactions.up.sql",
		"003_create_user_settings.up.sql",
	}
	for _, file := range files {
		sqlBytes, err := os.ReadFile(filepath.Join(migrationsDir, file))
		if err != nil {
			t.Fatalf("read migration %s: %v", file, err)
		}
		if _, err := testPool.Exec(ctx, string(sqlBytes)); err != nil {
			t.Fatalf("apply migration %s: %v", file, err)
		}
	}

	return db.New(testPool)
}
