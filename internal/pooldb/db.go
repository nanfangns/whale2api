package pooldb

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	_ "github.com/lib/pq"
	_ "modernc.org/sqlite"

	"whale2api/internal/config"
)

// DB stores gateway keys and their associated account pools.
type DB struct {
	sql *sqlStore
}

type sqlStore struct {
	*sql.DB
	driver string
}

type sqlTx struct {
	*sql.Tx
	store *sqlStore
}

// DefaultDatabasePath is used when WHALE2API_DATABASE_PATH is unset (local dev).
const DefaultDatabasePath = "docker-data/whale2api/whale2api.db"

func Connect(ctx context.Context, path string) (*DB, error) {
	path, err := resolveDatabasePath(path)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}
	return connect(ctx, "sqlite", sqliteDSN(path))
}

func ConnectPostgres(ctx context.Context, url string) (*DB, error) {
	url = strings.TrimSpace(url)
	if url == "" {
		return nil, fmt.Errorf("empty PostgreSQL connection URL")
	}
	return connect(ctx, "postgres", url)
}

func connect(ctx context.Context, driver, dsn string) (*DB, error) {
	sqlDB, err := sql.Open(driver, dsn)
	if err != nil {
		return nil, fmt.Errorf("open %s database: %w", driver, err)
	}
	maxOpen := 4
	if driver == "postgres" {
		maxOpen = 2
	}
	sqlDB.SetMaxOpenConns(maxOpen)
	sqlDB.SetMaxIdleConns(maxOpen)
	if err := sqlDB.PingContext(ctx); err != nil {
		_ = sqlDB.Close()
		return nil, fmt.Errorf("ping %s database: %w", driver, err)
	}
	db := &DB{sql: &sqlStore{DB: sqlDB, driver: driver}}
	if err := migrate(ctx, sqlDB, driver); err != nil {
		_ = sqlDB.Close()
		return nil, err
	}
	return db, nil
}

func ConnectFromEnv(ctx context.Context) (*DB, error) {
	if url := strings.TrimSpace(os.Getenv("WHALE2API_DATABASE_URL")); url != "" {
		return ConnectPostgres(ctx, url)
	}
	if parseBool(os.Getenv("WHALE2API_REQUIRE_DATABASE_URL")) {
		return nil, fmt.Errorf("WHALE2API_DATABASE_URL is required in this deployment")
	}
	path := databasePathFromEnv()
	if path == "" {
		return nil, nil
	}
	return Connect(ctx, path)
}

// DatabasePath returns a safe database description suitable for logs.
func DatabasePath() string {
	if strings.TrimSpace(os.Getenv("WHALE2API_DATABASE_URL")) != "" {
		return "PostgreSQL (WHALE2API_DATABASE_URL; credentials redacted)"
	}
	return databasePathFromEnv()
}

func DatabaseBackend() string {
	if strings.TrimSpace(os.Getenv("WHALE2API_DATABASE_URL")) != "" {
		return "postgres"
	}
	return "sqlite"
}

func (s *sqlStore) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return s.DB.ExecContext(ctx, s.bind(query), args...)
}

func (s *sqlStore) QueryContext(ctx context.Context, query string, args ...any) (*sql.Rows, error) {
	return s.DB.QueryContext(ctx, s.bind(query), args...)
}

func (s *sqlStore) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return s.DB.QueryRowContext(ctx, s.bind(query), args...)
}

func (s *sqlStore) BeginTx(ctx context.Context, opts *sql.TxOptions) (*sqlTx, error) {
	tx, err := s.DB.BeginTx(ctx, opts)
	if err != nil {
		return nil, err
	}
	return &sqlTx{Tx: tx, store: s}, nil
}

func (tx *sqlTx) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	return tx.Tx.ExecContext(ctx, tx.store.bind(query), args...)
}

func (tx *sqlTx) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	return tx.Tx.QueryRowContext(ctx, tx.store.bind(query), args...)
}

func (s *sqlStore) bind(query string) string {
	if s == nil || s.driver != "postgres" {
		return query
	}
	return rebindPostgres(query)
}

func databasePathFromEnv() string {
	if p := strings.TrimSpace(os.Getenv("WHALE2API_DATABASE_PATH")); p != "" {
		return p
	}
	return DefaultDatabasePath
}

func parseBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func resolveDatabasePath(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", fmt.Errorf("empty database path")
	}
	if !filepath.IsAbs(path) {
		if wd, err := os.Getwd(); err == nil {
			path = filepath.Join(wd, path)
		}
	}
	return filepath.Clean(path), nil
}

func sqliteDSN(path string) string {
	// WAL + busy_timeout help whale2api and poolui share one file.
	return "file:" + filepath.ToSlash(path) + "?_pragma=foreign_keys(1)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
}

func (db *DB) Close() {
	if db == nil || db.sql == nil {
		return
	}
	_ = db.sql.Close()
	db.sql = nil
}

func (db *DB) configured() error {
	if db == nil || db.sql == nil {
		return fmt.Errorf("pool db is not configured")
	}
	return nil
}

func maxAccountsPerKey() int {
	raw := strings.TrimSpace(os.Getenv("WHALE2API_POOL_MAX_ACCOUNTS_PER_KEY"))
	if raw == "" || raw == "0" {
		return 0
	}
	n, err := strconv.Atoi(raw)
	if err != nil || n < 0 {
		return 0
	}
	return n
}

func limitAccounts(accounts []config.Account) []config.Account {
	max := maxAccountsPerKey()
	if max <= 0 || len(accounts) <= max {
		return accounts
	}
	return accounts[:max]
}
