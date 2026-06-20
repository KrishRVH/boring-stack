package migrations

import "embed"

// FS contains the application Goose migrations. River's job-table migrations are
// intentionally run by River's own migrator in cmd/migrate.
//
//go:embed *.sql
var FS embed.FS
