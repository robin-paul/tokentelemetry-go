package scanner

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner/parsers"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
)

// Engine orchestrates discovery, worker concurrency, parsing, pricing, and batch transaction commits.
type Engine struct {
	db              *store.DB
	pricingEngine   *pricing.Engine
	registry        *parsers.Registry
	checkpoints     *CheckpointManager
	workerPoolSize  int
	batchTimeout    time.Duration
	batchSize       int
	taskQueue       chan string
	batchQueue      chan *models.Session
	onSessionEvent  func(eventType events.EventType, session *models.Session)
	onScanProgress  func(currentFile string, processed, total int)
	mu              sync.Mutex
	running         bool
	cancelWorkers   context.CancelFunc
	wg              sync.WaitGroup
}

// Config provides engine configuration options.
type Config struct {
	WorkerPoolSize int
	BatchTimeout   time.Duration
	BatchSize      int
	OnSessionEvent func(eventType events.EventType, session *models.Session)
	OnScanProgress func(currentFile string, processed, total int)
}

// NewEngine constructs a new scanner Engine.
func NewEngine(db *store.DB, pe *pricing.Engine, cfg Config) *Engine {
	workers := cfg.WorkerPoolSize
	if workers <= 0 {
		workers = min(runtime.NumCPU(), 8)
		if workers < 2 {
			workers = 2
		}
	}

	timeout := cfg.BatchTimeout
	if timeout <= 0 {
		timeout = 100 * time.Millisecond
	}

	batchSize := cfg.BatchSize
	if batchSize <= 0 {
		batchSize = 50
	}

	return &Engine{
		db:             db,
		pricingEngine:  pe,
		registry:       parsers.NewDefaultRegistry(),
		checkpoints:    NewCheckpointManager(db),
		workerPoolSize: workers,
		batchTimeout:   timeout,
		batchSize:      batchSize,
		taskQueue:      make(chan string, 1024),
		batchQueue:     make(chan *models.Session, 512),
		onSessionEvent: cfg.OnSessionEvent,
		onScanProgress: cfg.OnScanProgress,
	}
}

// GetRegistry returns the parser registry.
func (e *Engine) GetRegistry() *parsers.Registry {
	return e.registry
}

// Start launches the background worker pool and batch commit loop.
func (e *Engine) Start(ctx context.Context) {
	e.mu.Lock()
	if e.running {
		e.mu.Unlock()
		return
	}
	e.running = true

	workerCtx, cancel := context.WithCancel(ctx)
	e.cancelWorkers = cancel
	e.mu.Unlock()

	// 1. Launch Worker Pool
	for i := 0; i < e.workerPoolSize; i++ {
		e.wg.Add(1)
		go func(workerID int) {
			defer e.wg.Done()
			e.workerLoop(workerCtx)
		}(i)
	}

	// 2. Launch Batch Writer
	e.wg.Add(1)
	go func() {
		defer e.wg.Done()
		e.batchWriterLoop(workerCtx)
	}()
}

// Stop gracefully stops worker pool and flushes batch queues.
func (e *Engine) Stop() {
	e.mu.Lock()
	if !e.running {
		e.mu.Unlock()
		return
	}
	e.running = false
	if e.cancelWorkers != nil {
		e.cancelWorkers()
	}
	e.mu.Unlock()

	e.wg.Wait()
}

// EnqueueFile sends a file to the scanning pipeline.
func (e *Engine) EnqueueFile(filePath string) {
	select {
	case e.taskQueue <- filePath:
	default:
		// Queue full, process inline in a separate goroutine
		go func() {
			e.taskQueue <- filePath
		}()
	}
}

func (e *Engine) workerLoop(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case filePath, ok := <-e.taskQueue:
			if !ok {
				return
			}
			sess, err := e.ScanFile(ctx, filePath)
			if err != nil || sess == nil {
				continue
			}
			select {
			case <-ctx.Done():
				return
			case e.batchQueue <- sess:
			}
		}
	}
}

func (e *Engine) batchWriterLoop(ctx context.Context) {
	ticker := time.NewTicker(e.batchTimeout)
	defer ticker.Stop()

	var batch []*models.Session

	flush := func() {
		if len(batch) == 0 {
			return
		}
		toCommit := batch
		batch = nil
		e.commitBatch(context.Background(), toCommit)
	}

	for {
		select {
		case <-ctx.Done():
			// Flush remaining items
			flush()
			return
		case sess := <-e.batchQueue:
			batch = append(batch, sess)
			if len(batch) >= e.batchSize {
				flush()
			}
		case <-ticker.C:
			flush()
		}
	}
}

func (e *Engine) commitBatch(ctx context.Context, sessions []*models.Session) {
	if len(sessions) == 0 {
		return
	}

	affectedDates := make(map[string]bool)
	var committed []*models.Session

	for _, s := range sessions {
		// Save session, message turns, and subagent runs
		if err := e.db.SaveSessionWithTurnsAndSubagents(ctx, s); err != nil {
			continue
		}

		committed = append(committed, s)

		// Track date for summary rollup
		dateStr := s.StartTime.Format("2006-01-02")
		if dateStr != "" && dateStr != "0001-01-01" {
			affectedDates[dateStr] = true
		} else {
			affectedDates[time.Now().Format("2006-01-02")] = true
		}
	}

	// Rollup affected daily summaries BEFORE broadcasting events
	for date := range affectedDates {
		_ = e.db.RollupDailySummariesForDate(ctx, date)
	}

	// Broadcast events after database state and daily summaries are committed
	if e.onSessionEvent != nil {
		for _, s := range committed {
			e.onSessionEvent(events.EventSessionCreated, s)
		}
	}
}

// ScanFile inspects, parses, costs, and returns a single session file.
func (e *Engine) ScanFile(ctx context.Context, filePath string) (*models.Session, error) {
	state, err := GetFileState(filePath)
	if err != nil {
		return nil, err
	}

	shouldScan, cp, err := e.checkpoints.ShouldScan(ctx, state)
	if err != nil || !shouldScan {
		return nil, err
	}

	parser := e.registry.Detect(filePath)
	if parser == nil {
		return nil, nil
	}

	f, err := os.Open(filePath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	startOffset := int64(0)
	// If checkpoint exists and file was appended
	if cp != nil && state.FileSize >= cp.FileSize {
		// For JSONL files, we can resume from byteOffset if supported
		// For full document JSON/SQLite files, reparse from 0
	}

	parsed, endOffset, err := parser.Parse(f, startOffset)
	if err != nil {
		return nil, err
	}
	if parsed == nil {
		return nil, nil
	}

	projectName := parsed.ProjectName
	if projectName == "" {
		projectName = parsers.ExtractProjectName(filePath)
	}

	// Build models.Session
	sess := &models.Session{
		ID:                  parsed.ID,
		SessionID:           parsed.SessionID,
		AgentName:           parsed.AgentName,
		ProjectName:         projectName,
		FilePath:            filePath,
		StartTime:           parsed.StartTime,
		EndTime:             parsed.EndTime,
		DurationSeconds:     parsed.DurationSeconds,
		ModelRaw:            parsed.Model,
		InputTokens:         parsed.TotalUsage.InputTokens,
		OutputTokens:        parsed.TotalUsage.OutputTokens,
		CacheReadTokens:     parsed.TotalUsage.CacheReadTokens,
		CacheCreationTokens: parsed.TotalUsage.CacheCreationTokens,
		HardwareProfile:     parsed.HardwareProfile,
		Status:              parsed.Status,
		GitBranch:           parsed.GitBranch,
		IsSubagent:          parsed.IsSubagent,
		ParentSessionID:     parsed.ParentSessionID,
		SubagentType:        parsed.SubagentType,
		SubagentRuns:        parsed.SubagentRuns,
	}

	if sess.HardwareProfile == "" {
		sess.HardwareProfile = "default"
	}
	if sess.Status == "" {
		sess.Status = "completed"
	}

	// Convert turns
	sess.Turns = make([]models.MessageTurn, len(parsed.Turns))
	for i, t := range parsed.Turns {
		sess.Turns[i] = models.MessageTurn{
			ID:                  fmt.Sprintf("%s:%d", sess.ID, t.Index),
			SessionID:           sess.ID,
			TurnIndex:           t.Index,
			Timestamp:           t.Timestamp,
			Role:                t.Role,
			ModelName:           t.Model,
			InputTokens:         t.Usage.InputTokens,
			OutputTokens:        t.Usage.OutputTokens,
			CacheReadTokens:     t.Usage.CacheReadTokens,
			CacheCreationTokens: t.Usage.CacheCreationTokens,
			ToolsInvoked:        t.Tools,
		}
	}

	// Run Offline Pricing Engine
	overrides, _ := e.db.GetPricingOverrides(ctx)
	e.pricingEngine.CostSession(ctx, sess, parsed.Endpoint, parsed.Provider, overrides)

	// Update Checkpoint
	_ = e.checkpoints.UpdateCheckpoint(ctx, filePath, state, endOffset, len(sess.Turns), "")

	return sess, nil
}

// DiscoverDefaultRoots discovers standard agent directories across the user's home folder.
func DiscoverDefaultRoots() []string {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil
	}

	candidates := []string{
		filepath.Join(home, ".claude", "projects"),
		filepath.Join(home, ".gemini", "antigravity-cli", "brain"),
		filepath.Join(home, ".gemini", "antigravity-ide", "brain"),
		filepath.Join(home, ".gemini", "transcripts"),
		filepath.Join(home, ".gemini", "tmp"),
		filepath.Join(home, ".codex", "sessions"),
		filepath.Join(home, ".cursor", "projects"),
		filepath.Join(home, ".copilot", "session-state"),
		filepath.Join(home, ".local", "share", "opencode"),
		filepath.Join(home, ".hermes", "telemetry"),
		filepath.Join(home, ".hermes"),
		filepath.Join(home, ".grok", "sessions"),
		filepath.Join(home, ".pi", "agent", "sessions"),
		filepath.Join(home, ".dsh", "sessions"),
		filepath.Join(home, ".muse", "sessions"),
		filepath.Join(home, ".prime", "sessions"),
		filepath.Join(home, ".qwen", "projects"),
		filepath.Join(home, ".cline", "data"),
	}

	var existing []string
	for _, c := range candidates {
		if fi, err := os.Stat(c); err == nil && (fi.IsDir() || strings.HasSuffix(c, ".db")) {
			existing = append(existing, c)
		}
	}

	return existing
}

// ScanRoots walks the given directories and scans all detected agent transcripts.
func (e *Engine) ScanRoots(ctx context.Context, roots []string) error {
	var filesToScan []string

	for _, root := range roots {
		fi, err := os.Stat(root)
		if err != nil {
			continue
		}

		if !fi.IsDir() {
			if e.registry.Detect(root) != nil {
				filesToScan = append(filesToScan, root)
			}
			continue
		}

		_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				// Skip node_modules, .git, cache
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "dist" || name == ".cache" {
					return filepath.SkipDir
				}
				return nil
			}

			if e.registry.Detect(path) != nil {
				filesToScan = append(filesToScan, path)
			}
			return nil
		})
	}

	total := len(filesToScan)
	for i, file := range filesToScan {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if e.onScanProgress != nil {
				e.onScanProgress(file, i+1, total)
			}
			e.EnqueueFile(file)
		}
	}

	return nil
}
