package scanner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"io"
	"os"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

// CheckpointManager coordinates file state checking and checkpoint updates.
type CheckpointManager struct {
	db *store.DB
}

// NewCheckpointManager creates a new CheckpointManager instance.
func NewCheckpointManager(db *store.DB) *CheckpointManager {
	return &CheckpointManager{db: db}
}

// FileState holds the current filesystem state of a file.
type FileState struct {
	FilePath     string
	LastModified time.Time
	FileSize     int64
}

// GetFileState inspects the file on disk.
func GetFileState(filePath string) (*FileState, error) {
	fi, err := os.Stat(filePath)
	if err != nil {
		return nil, err
	}
	return &FileState{
		FilePath:     filePath,
		LastModified: fi.ModTime().UTC(),
		FileSize:     fi.Size(),
	}, nil
}

// ShouldScan determines if the file needs to be scanned or rescanned.
func (cm *CheckpointManager) ShouldScan(ctx context.Context, state *FileState) (bool, *store.ScannerCheckpoint, error) {
	if cm == nil || cm.db == nil {
		return true, nil, nil
	}
	cp, err := cm.db.GetCheckpoint(ctx, state.FilePath)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			return true, nil, nil
		}
		return false, nil, err
	}

	// If file size or last modified time changed, it must be scanned
	if !state.LastModified.Equal(cp.LastModified) || state.FileSize != cp.FileSize {
		return true, cp, nil
	}

	return false, cp, nil
}

// UpdateCheckpoint updates the recorded checkpoint in SQLite.
func (cm *CheckpointManager) UpdateCheckpoint(ctx context.Context, filePath string, state *FileState, byteOffset int64, lineNum int, fileHash string) error {
	if cm == nil || cm.db == nil {
		return nil
	}
	cp := &store.ScannerCheckpoint{
		FilePath:     filePath,
		LastModified: state.LastModified,
		FileSize:     state.FileSize,
		ByteOffset:   byteOffset,
		LineNumber:   lineNum,
		FileHash:     fileHash,
	}
	return cm.db.UpsertCheckpoint(ctx, cp)
}

// ComputeFileHash calculates SHA256 hash of a file or reader.
func ComputeFileHash(r io.Reader) (string, error) {
	h := sha256.New()
	if _, err := io.Copy(h, r); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
