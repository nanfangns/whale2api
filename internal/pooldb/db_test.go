package pooldb

import (
	"context"
	"path/filepath"
	"strings"
	"testing"
)

func TestConnectSQLiteMigrations(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.db")
	db, err := Connect(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	exists, err := db.GatewayKeyExists(context.Background(), "missing")
	if err != nil {
		t.Fatal(err)
	}
	if exists {
		t.Fatal("expected missing key")
	}
}

func TestIsUniqueViolationSQLite(t *testing.T) {
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "uniq.db")
	db, err := Connect(ctx, path)
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if err := db.CreateGatewayKey(ctx, "sk-dup", "a", ""); err != nil {
		t.Fatal(err)
	}
	err = db.CreateGatewayKey(ctx, "sk-dup", "b", "")
	if !IsUniqueViolation(err) {
		t.Fatalf("expected unique violation, got %v", err)
	}
}

func TestConnectFromEnvRequiresPostgresURL(t *testing.T) {
	t.Setenv("WHALE2API_DATABASE_URL", "")
	t.Setenv("WHALE2API_REQUIRE_DATABASE_URL", "true")
	if _, err := ConnectFromEnv(context.Background()); err == nil || !strings.Contains(err.Error(), "WHALE2API_DATABASE_URL is required") {
		t.Fatalf("expected missing PostgreSQL URL error, got %v", err)
	}
}

func TestDatabasePathRedactsPostgresCredentials(t *testing.T) {
	t.Setenv("WHALE2API_DATABASE_URL", "postgres://user:secret@example.invalid:5432/pool")
	if got := DatabasePath(); strings.Contains(got, "secret") {
		t.Fatalf("DatabasePath leaked credentials: %q", got)
	}
	if got := DatabaseBackend(); got != "postgres" {
		t.Fatalf("DatabaseBackend() = %q, want postgres", got)
	}
}
