package pooldb

import (
	"errors"
	"strings"
)

// IsUniqueViolation reports duplicate key errors from supported SQL drivers.
func IsUniqueViolation(err error) bool {
	if err == nil {
		return false
	}
	var sqlStateErr interface{ SQLState() string }
	if errors.As(err, &sqlStateErr) && sqlStateErr.SQLState() == "23505" {
		return true
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "unique constraint failed") ||
		strings.Contains(msg, "constraint failed: unique") ||
		strings.Contains(msg, "duplicate key value violates unique constraint")
}
