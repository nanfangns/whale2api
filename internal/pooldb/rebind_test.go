package pooldb

import "testing"

func TestRebindPostgresPlaceholders(t *testing.T) {
	input := `SELECT ?, '?', "?", ? -- ?
/* ? */ WHERE id = ?`
	want := `SELECT $1, '?', "?", $2 -- ?
/* ? */ WHERE id = $3`
	if got := rebindPostgres(input); got != want {
		t.Fatalf("rebindPostgres() = %q, want %q", got, want)
	}
}

func TestRebindPostgresSQLiteTimestamp(t *testing.T) {
	input := `UPDATE jobs SET updated_at = datetime('now') WHERE id = ?`
	want := `UPDATE jobs SET updated_at = ` + postgresNowExpression + ` WHERE id = $1`
	if got := rebindPostgres(input); got != want {
		t.Fatalf("rebindPostgres() = %q, want %q", got, want)
	}
}
