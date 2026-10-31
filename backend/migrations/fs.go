package migrations

import "embed"

// Files is the SQL migration set applied on API startup.
//
//go:embed *.sql
var Files embed.FS
