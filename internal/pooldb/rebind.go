package pooldb

import (
	"fmt"
	"strings"
)

func rebindPostgres(query string) string {
	query = strings.ReplaceAll(query, "datetime('now')", postgresNowExpression)
	var out strings.Builder
	out.Grow(len(query) + 16)
	argument := 0
	inSingleQuote := false
	inDoubleQuote := false
	inLineComment := false
	inBlockComment := false

	for i := 0; i < len(query); i++ {
		ch := query[i]
		next := byte(0)
		if i+1 < len(query) {
			next = query[i+1]
		}

		switch {
		case inLineComment:
			out.WriteByte(ch)
			if ch == '\n' {
				inLineComment = false
			}
		case inBlockComment:
			out.WriteByte(ch)
			if ch == '*' && next == '/' {
				out.WriteByte(next)
				i++
				inBlockComment = false
			}
		case inSingleQuote:
			out.WriteByte(ch)
			if ch == '\'' {
				if next == '\'' {
					out.WriteByte(next)
					i++
				} else {
					inSingleQuote = false
				}
			}
		case inDoubleQuote:
			out.WriteByte(ch)
			if ch == '"' {
				if next == '"' {
					out.WriteByte(next)
					i++
				} else {
					inDoubleQuote = false
				}
			}
		case ch == '-' && next == '-':
			out.WriteString("--")
			i++
			inLineComment = true
		case ch == '/' && next == '*':
			out.WriteString("/*")
			i++
			inBlockComment = true
		case ch == '\'':
			out.WriteByte(ch)
			inSingleQuote = true
		case ch == '"':
			out.WriteByte(ch)
			inDoubleQuote = true
		case ch == '?':
			argument++
			out.WriteString(fmt.Sprintf("$%d", argument))
		default:
			out.WriteByte(ch)
		}
	}

	return out.String()
}

const postgresNowExpression = `to_char(CURRENT_TIMESTAMP AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS.US"Z"')`
