package watcher

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
)

// Watcher monitors filesystem directories for agent transcript updates with event debouncing.
type Watcher struct {
	fsWatcher     *fsnotify.Watcher
	scanner       *scanner.Engine
	debounceTime  time.Duration
	debounceTimers map[string]*time.Timer
	timerMu       sync.Mutex
	watchedPaths  map[string]bool
	pathsMu       sync.RWMutex
	running       bool
	runMu         sync.Mutex
	cancel        context.CancelFunc
	wg            sync.WaitGroup
}

// Config provides Watcher configuration options.
type Config struct {
	DebounceDuration time.Duration
}

// NewWatcher creates a new Watcher instance attached to a scanner engine.
func NewWatcher(eng *scanner.Engine, cfg Config) (*Watcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create fsnotify watcher: %w", err)
	}

	debounce := cfg.DebounceDuration
	if debounce <= 0 {
		debounce = 250 * time.Millisecond
	}

	return &Watcher{
		fsWatcher:      fsw,
		scanner:        eng,
		debounceTime:   debounce,
		debounceTimers: make(map[string]*time.Timer),
		watchedPaths:   make(map[string]bool),
	}, nil
}

// AddRoot recursively adds a root directory and all its child directories to the watch list.
func (w *Watcher) AddRoot(root string) error {
	fi, err := os.Stat(root)
	if err != nil {
		return err
	}

	if !fi.IsDir() {
		return w.addSinglePath(root)
	}

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "dist" || name == ".cache" {
				return filepath.SkipDir
			}
			_ = w.addSinglePath(path)
		}
		return nil
	})
}

func (w *Watcher) addSinglePath(path string) error {
	w.pathsMu.Lock()
	defer w.pathsMu.Unlock()

	if w.watchedPaths[path] {
		return nil
	}

	if err := w.fsWatcher.Add(path); err != nil {
		return err
	}
	w.watchedPaths[path] = true
	return nil
}

// Start begins processing fsnotify events in background goroutines.
func (w *Watcher) Start(ctx context.Context) {
	w.runMu.Lock()
	if w.running {
		w.runMu.Unlock()
		return
	}
	w.running = true

	watchCtx, cancel := context.WithCancel(ctx)
	w.cancel = cancel
	w.runMu.Unlock()

	w.wg.Add(1)
	go func() {
		defer w.wg.Done()
		w.eventLoop(watchCtx)
	}()
}

// Stop terminates the watcher and cancels pending debounce timers.
func (w *Watcher) Stop() error {
	w.runMu.Lock()
	if !w.running {
		w.runMu.Unlock()
		return nil
	}
	w.running = false
	if w.cancel != nil {
		w.cancel()
	}
	w.runMu.Unlock()

	w.timerMu.Lock()
	for _, t := range w.debounceTimers {
		t.Stop()
	}
	w.debounceTimers = make(map[string]*time.Timer)
	w.timerMu.Unlock()

	err := w.fsWatcher.Close()
	w.wg.Wait()
	return err
}

func (w *Watcher) eventLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleFSEvent(event)
		case _, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
		}
	}
}

func (w *Watcher) handleFSEvent(event fsnotify.Event) {
	filePath := event.Name

	// Dynamically register newly created directories
	if event.Has(fsnotify.Create) {
		if fi, err := os.Stat(filePath); err == nil && fi.IsDir() {
			_ = w.AddRoot(filePath)
			return
		}
	}

	// Check if this event represents file modification / write / create / rename
	if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) || event.Has(fsnotify.Rename) {
		// Only forward files matching known parser patterns
		if w.scanner != nil && w.scanner.GetRegistry().Detect(filePath) != nil {
			w.debounceFile(filePath)
		}
	}
}

func (w *Watcher) debounceFile(filePath string) {
	w.timerMu.Lock()
	defer w.timerMu.Unlock()

	if t, exists := w.debounceTimers[filePath]; exists {
		t.Stop()
	}

	w.debounceTimers[filePath] = time.AfterFunc(w.debounceTime, func() {
		w.timerMu.Lock()
		delete(w.debounceTimers, filePath)
		w.timerMu.Unlock()

		if w.scanner != nil {
			w.scanner.EnqueueFile(filePath)
		}
	})
}
