// Package migrations embeds the versioned SQL migration files so they ship
// inside the compiled binary — no separate migrations directory needs to
// be copied into the Docker image or deployed alongside it. See
// internal/platform/migrator for what runs these.
package migrations

import "embed"

//go:embed *.sql
var FS embed.FS
