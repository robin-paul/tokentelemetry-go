package store

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"sort"
	"strconv"
	"strings"
	"time"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

// Migrate executes all pending SQL migrations in ascending order.
func (d *DB) Migrate(ctx context.Context) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		// 1. Ensure migrations tracker exists
		_, err := tx.ExecContext(ctx, `
			CREATE TABLE IF NOT EXISTS schema_migrations (
				version INTEGER PRIMARY KEY,
				applied_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
			);
		`)
		if err != nil {
			return fmt.Errorf("failed to create schema_migrations table: %w", err)
		}

		// 2. Query applied migrations
		rows, err := tx.QueryContext(ctx, `SELECT version FROM schema_migrations;`)
		if err != nil {
			return fmt.Errorf("failed to query applied migrations: %w", err)
		}
		defer rows.Close()

		applied := make(map[int]bool)
		for rows.Next() {
			var v int
			if err := rows.Scan(&v); err != nil {
				return fmt.Errorf("failed to scan migration version: %w", err)
			}
			applied[v] = true
		}
		if err := rows.Err(); err != nil {
			return err
		}

		// 3. Read embedded migration files
		entries, err := fs.ReadDir(migrationsFS, "migrations")
		if err != nil {
			return fmt.Errorf("failed to read migrations dir: %w", err)
		}

		type migrationFile struct {
			version  int
			filename string
		}

		var files []migrationFile
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
				continue
			}
			parts := strings.Split(entry.Name(), "_")
			if len(parts) == 0 {
				continue
			}
			v, err := strconv.Atoi(parts[0])
			if err != nil {
				continue
			}
			files = append(files, migrationFile{
				version:  v,
				filename: entry.Name(),
			})
		}

		sort.Slice(files, func(i, j int) bool {
			return files[i].version < files[j].version
		})

		// 4. Apply pending migrations
		for _, file := range files {
			if applied[file.version] {
				continue
			}

			content, err := fs.ReadFile(migrationsFS, "migrations/"+file.filename)
			if err != nil {
				return fmt.Errorf("failed to read migration file %s: %w", file.filename, err)
			}

			// Execute script statements
			if _, err := tx.ExecContext(ctx, string(content)); err != nil {
				return fmt.Errorf("migration %s failed: %w", file.filename, err)
			}

			// Record migration version
			if _, err := tx.ExecContext(ctx, `INSERT INTO schema_migrations (version, applied_at) VALUES (?, ?);`, file.version, time.Now().UTC()); err != nil {
				return fmt.Errorf("failed to record migration %d: %w", file.version, err)
			}
		}

		return nil
	})
}
