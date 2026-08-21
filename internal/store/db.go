package store

import (
	"context"
	"database/sql"
	"fmt"
	"runtime"
	"strings"
	"sync"
	"time"

	_ "modernc.org/sqlite"
)

// DB wraps read and write connection pools to SQLite.
type DB struct {
	writerDB *sql.DB
	readerDB *sql.DB
	writeMu  sync.Mutex
	isShared bool
}

// Open initializes and configures a SQLite connection pool with WAL mode and pragmas.
func Open(dsn string) (*DB, error) {
	isMemory := dsn == ":memory:" || strings.Contains(dsn, "mode=memory")

	// Ensure pragmas in connection string
	connStr := dsn
	if !strings.Contains(connStr, "_pragma") && !isMemory {
		separator := "?"
		if strings.Contains(connStr, "?") {
			separator = "&"
		}
		connStr = fmt.Sprintf("%s%s_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)", connStr, separator)
	}

	if isMemory {
		// For in-memory DB (such as during tests), use a shared single DB pool
		db, err := sql.Open("sqlite", connStr)
		if err != nil {
			return nil, fmt.Errorf("failed to open in-memory sqlite: %w", err)
		}
		db.SetMaxOpenConns(1)
		db.SetMaxIdleConns(1)
		db.SetConnMaxLifetime(0)

		storeDB := &DB{
			writerDB: db,
			readerDB: db,
			isShared: true,
		}
		if err := storeDB.applyPragmas(db, isMemory); err != nil {
			_ = db.Close()
			return nil, err
		}
		return storeDB, nil
	}

	// 1. Dedicated Single-Writer Connection
	writerDB, err := sql.Open("sqlite", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite writer: %w", err)
	}
	writerDB.SetMaxOpenConns(1)
	writerDB.SetMaxIdleConns(1)
	writerDB.SetConnMaxLifetime(0)

	// 2. Multi-Reader Pool
	readerDB, err := sql.Open("sqlite", connStr)
	if err != nil {
		_ = writerDB.Close()
		return nil, fmt.Errorf("failed to open sqlite reader: %w", err)
	}
	maxReaders := max(4, runtime.NumCPU()*2)
	maxIdleReaders := max(2, runtime.NumCPU())
	readerDB.SetMaxOpenConns(maxReaders)
	readerDB.SetMaxIdleConns(maxIdleReaders)
	readerDB.SetConnMaxLifetime(0)

	storeDB := &DB{
		writerDB: writerDB,
		readerDB: readerDB,
		isShared: false,
	}

	if err := storeDB.applyPragmas(writerDB, false); err != nil {
		_ = storeDB.Close()
		return nil, err
	}
	if err := storeDB.applyPragmas(readerDB, false); err != nil {
		_ = storeDB.Close()
		return nil, err
	}

	return storeDB, nil
}

func (d *DB) applyPragmas(db *sql.DB, isMemory bool) error {
	pragmas := []string{
		"PRAGMA busy_timeout = 5000;",
		"PRAGMA foreign_keys = ON;",
	}
	if !isMemory {
		pragmas = append(pragmas,
			"PRAGMA journal_mode = WAL;",
			"PRAGMA synchronous = NORMAL;",
			"PRAGMA cache_size = -64000;", // 64MB memory cache
			"PRAGMA temp_store = MEMORY;",
		)
	}
	for _, p := range pragmas {
		if _, err := db.Exec(p); err != nil {
			return fmt.Errorf("pragma %q failed: %w", p, err)
		}
	}
	return nil
}

// Writer returns the write-capable database handle.
func (d *DB) Writer() *sql.DB {
	return d.writerDB
}

// Reader returns the read-only database pool handle.
func (d *DB) Reader() *sql.DB {
	return d.readerDB
}

// WithTx executes a write transaction with serialized locking to prevent database busy locks.
func (d *DB) WithTx(ctx context.Context, fn func(tx *sql.Tx) error) error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	tx, err := d.writerDB.BeginTx(ctx, &sql.TxOptions{Isolation: sql.LevelSerializable})
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	if err := fn(tx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Close gracefully closes the underlying database connection pools.
func (d *DB) Close() error {
	d.writeMu.Lock()
	defer d.writeMu.Unlock()

	var firstErr error
	if d.isShared {
		return d.writerDB.Close()
	}
	if d.writerDB != nil {
		if err := d.writerDB.Close(); err != nil {
			firstErr = err
		}
	}
	if d.readerDB != nil {
		if err := d.readerDB.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

// Ping verifies database connectivity.
func (d *DB) Ping(ctx context.Context) error {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()
	if err := d.writerDB.PingContext(ctx); err != nil {
		return fmt.Errorf("writer ping failed: %w", err)
	}
	if err := d.readerDB.PingContext(ctx); err != nil {
		return fmt.Errorf("reader ping failed: %w", err)
	}
	return nil
}
