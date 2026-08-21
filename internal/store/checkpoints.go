package store

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// ScannerCheckpoint tracks the state and byte offset of a scanned log file.
type ScannerCheckpoint struct {
	FilePath     string    `json:"file_path"`
	LastModified time.Time `json:"last_modified"`
	FileSize     int64     `json:"file_size"`
	ByteOffset   int64     `json:"byte_offset"`
	LineNumber   int       `json:"line_number"`
	FileHash     string    `json:"file_hash"`
}

// GetCheckpoint retrieves the scanner checkpoint for a specific file.
func (d *DB) GetCheckpoint(ctx context.Context, filePath string) (*ScannerCheckpoint, error) {
	query := `
	SELECT file_path, last_modified, file_size, byte_offset, line_number, file_hash
	FROM scanner_checkpoints
	WHERE file_path = ?;
	`
	var cp ScannerCheckpoint
	err := d.readerDB.QueryRowContext(ctx, query, filePath).Scan(
		&cp.FilePath, &cp.LastModified, &cp.FileSize,
		&cp.ByteOffset, &cp.LineNumber, &cp.FileHash,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get checkpoint: %w", err)
	}
	return &cp, nil
}

// UpsertCheckpoint inserts or updates a scanner checkpoint record.
func (d *DB) UpsertCheckpoint(ctx context.Context, cp *ScannerCheckpoint) error {
	query := `
	INSERT INTO scanner_checkpoints (
		file_path, last_modified, file_size, byte_offset, line_number, file_hash
	) VALUES (?, ?, ?, ?, ?, ?)
	ON CONFLICT(file_path) DO UPDATE SET
		last_modified = excluded.last_modified,
		file_size = excluded.file_size,
		byte_offset = excluded.byte_offset,
		line_number = excluded.line_number,
		file_hash = excluded.file_hash;
	`
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, query,
			cp.FilePath, cp.LastModified, cp.FileSize,
			cp.ByteOffset, cp.LineNumber, cp.FileHash,
		)
		return err
	})
}

// ListCheckpoints lists all stored scanner checkpoints.
func (d *DB) ListCheckpoints(ctx context.Context) ([]ScannerCheckpoint, error) {
	query := `
	SELECT file_path, last_modified, file_size, byte_offset, line_number, file_hash
	FROM scanner_checkpoints
	ORDER BY last_modified DESC;
	`
	rows, err := d.readerDB.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to list checkpoints: %w", err)
	}
	defer rows.Close()

	var list []ScannerCheckpoint
	for rows.Next() {
		var cp ScannerCheckpoint
		if err := rows.Scan(
			&cp.FilePath, &cp.LastModified, &cp.FileSize,
			&cp.ByteOffset, &cp.LineNumber, &cp.FileHash,
		); err != nil {
			return nil, fmt.Errorf("failed to scan checkpoint: %w", err)
		}
		list = append(list, cp)
	}
	return list, rows.Err()
}

// DeleteCheckpoint removes a checkpoint for a deleted file.
func (d *DB) DeleteCheckpoint(ctx context.Context, filePath string) error {
	return d.WithTx(ctx, func(tx *sql.Tx) error {
		_, err := tx.ExecContext(ctx, `DELETE FROM scanner_checkpoints WHERE file_path = ?;`, filePath)
		return err
	})
}
