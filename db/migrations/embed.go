// Package migrations embeds the SQL migration files so they can be applied by
// the cmd/migrate binary without shipping the .sql files separately.
package migrations

import "embed"

// FS holds every *.sql migration in this directory.
//
//go:embed *.sql
var FS embed.FS
