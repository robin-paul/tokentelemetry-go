package collector

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/client"
	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
	"github.com/robin-paul/tokentelemetry-go/internal/watcher"
)

// ScanSummary aggregates metrics from a one-off directory scan.
type ScanSummary struct {
	TotalFiles       int           `json:"total_files"`
	ParsedSessions   int           `json:"parsed_sessions"`
	TotalTurns       int           `json:"total_turns"`
	TotalInputTokens int64         `json:"total_input_tokens"`
	TotalOutputTokens int64        `json:"total_output_tokens"`
	TotalCacheTokens int64         `json:"total_cache_tokens"`
	TotalCostUSD     float64       `json:"total_cost_usd"`
	Duration         time.Duration `json:"duration"`
	AcceptedSessions int           `json:"accepted_sessions"`
	AcceptedTurns    int           `json:"accepted_turns"`
	Errors           []string      `json:"errors,omitempty"`
}

// HubHealth contains connectivity status details for the remote Hub.
type HubHealth struct {
	Status        string        `json:"status"`
	HubURL        string        `json:"hub_url"`
	Latency       time.Duration `json:"latency"`
	ServerVersion string        `json:"server_version,omitempty"`
	Error         string        `json:"error,omitempty"`
}

// Pipeline coordinates scanning, local pricing, TUI/log presentation, and remote Hub ingestion.
type Pipeline struct {
	cfg           *Config
	sink          EventSink
	pricingEngine *pricing.Engine
	scannerEngine *scanner.Engine
	client        *client.Client
	buffer        *client.Buffer
	watcher       *watcher.Watcher
	reconciler    *watcher.Reconciler
	running       bool
	mu            sync.Mutex
}

// NewPipeline initializes a new Collector Pipeline with all sub-components.
func NewPipeline(cfg *Config, sink EventSink) (*Pipeline, error) {
	if cfg == nil {
		cfg = DefaultConfig()
	}
	if sink == nil {
		sink = NewConsoleSink(os.Stdout)
	}

	pe, err := pricing.NewEngine()
	if err != nil {
		return nil, fmt.Errorf("failed to initialize pricing engine: %w", err)
	}

	cliCfg := client.Config{
		HubURL:         cfg.HubURL,
		AuthToken:      cfg.AuthToken,
		MachineID:      cfg.MachineID,
		ClientVersion:  "2.0.0",
		MaxRetries:     cfg.MaxRetries,
		BaseRetryDelay: 500 * time.Millisecond,
		MaxRetryDelay:  15 * time.Second,
		Timeout:        time.Duration(cfg.TimeoutSec) * time.Second,
	}
	cli := client.NewClient(cliCfg)

	bufCfg := client.BufferConfig{
		BatchSize:     cfg.BatchSize,
		FlushInterval: time.Duration(cfg.FlushMS) * time.Millisecond,
		MaxQueueSize:  5000,
	}
	buf := client.NewBuffer(cli, bufCfg)

	p := &Pipeline{
		cfg:           cfg,
		sink:          sink,
		pricingEngine: pe,
		client:        cli,
		buffer:        buf,
	}

	buf.SetCallback(func(resp *models.IngestionResponse, err error) {
		if err != nil {
			p.sink.OnError(fmt.Errorf("hub ingestion error: %w", err))
		}
	})

	scannerCfg := scanner.Config{
		WorkerPoolSize: 4,
		BatchTimeout:   time.Duration(cfg.FlushMS) * time.Millisecond,
		BatchSize:      cfg.BatchSize,
		OnSessionEvent: func(eventType events.EventType, session *models.Session) {
			p.sink.OnSession(session)
			for i := range session.Turns {
				p.sink.OnTurn(&session.Turns[i], session)
			}
			if p.buffer != nil {
				p.buffer.Enqueue(session)
			}
		},
	}

	// Scanner engine with pure in-memory operation (no local DB on client)
	p.scannerEngine = scanner.NewEngine(nil, pe, scannerCfg)

	w, err := watcher.NewWatcher(p.scannerEngine, watcher.Config{
		DebounceDuration: 250 * time.Millisecond,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create watcher: %w", err)
	}
	p.watcher = w

	p.reconciler = watcher.NewReconciler(p.scannerEngine, watcher.ReconcilerConfig{
		Interval: 60 * time.Second,
		Roots:    cfg.ScanRoots,
	})

	return p, nil
}

// Config returns the pipeline configuration.
func (p *Pipeline) Config() *Config {
	return p.cfg
}

// Sink returns the attached EventSink.
func (p *Pipeline) Sink() EventSink {
	return p.sink
}

// Scanner returns the underlying scanner engine.
func (p *Pipeline) Scanner() *scanner.Engine {
	return p.scannerEngine
}

// Pricing returns the offline pricing engine.
func (p *Pipeline) Pricing() *pricing.Engine {
	return p.pricingEngine
}

// Start begins live watching and periodic reconciliation.
func (p *Pipeline) Start(ctx context.Context) error {
	p.mu.Lock()
	if p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = true
	p.mu.Unlock()

	p.buffer.Start(ctx)
	p.scannerEngine.Start(ctx)

	// Add all configured scan roots to the watcher
	activeRoots := 0
	for _, root := range p.cfg.ScanRoots {
		if fi, err := os.Stat(root); err == nil && fi.IsDir() {
			if err := p.watcher.AddRoot(root); err == nil {
				activeRoots++
			}
		}
	}

	p.watcher.Start(ctx)
	p.reconciler.Start(ctx)

	p.sink.OnStatus(fmt.Sprintf("TokenTelemetry Collector running. Watching %d root directories. Hub: %s", activeRoots, p.cfg.HubURL))
	return nil
}

// Stop gracefully stops watching, flushes transmission buffers, and closes the sink.
func (p *Pipeline) Stop() error {
	p.mu.Lock()
	if !p.running {
		p.mu.Unlock()
		return nil
	}
	p.running = false
	p.mu.Unlock()

	p.sink.OnStatus("Stopping collector...")
	p.reconciler.Stop()
	_ = p.watcher.Stop()
	p.scannerEngine.Stop()

	flushCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	p.buffer.Close(flushCtx)

	_ = p.sink.Close()
	return nil
}

// ScanOnce performs a synchronous one-off sweep of roots, parsing sessions and optionally streaming them to the Hub.
func (p *Pipeline) ScanOnce(ctx context.Context, roots []string, dryRun bool) (*ScanSummary, error) {
	startTime := time.Now()
	if len(roots) == 0 {
		roots = p.cfg.ScanRoots
	}

	var filesToScan []string
	registry := p.scannerEngine.GetRegistry()

	for _, root := range roots {
		fi, err := os.Stat(root)
		if err != nil {
			continue
		}

		if !fi.IsDir() {
			if registry.Detect(root) != nil {
				filesToScan = append(filesToScan, root)
			}
			continue
		}

		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "dist" || name == ".cache" {
					return filepath.SkipDir
				}
				return nil
			}
			if registry.Detect(path) != nil {
				filesToScan = append(filesToScan, path)
			}
			return nil
		})
	}

	summary := &ScanSummary{
		TotalFiles: len(filesToScan),
	}

	var sessionsToSend []models.Session

	for _, file := range filesToScan {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		sess, err := p.scannerEngine.ScanFile(ctx, file)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("%s: %v", filepath.Base(file), err))
			continue
		}
		if sess == nil {
			continue
		}

		sess.MachineID = p.cfg.MachineID
		summary.ParsedSessions++
		summary.TotalTurns += len(sess.Turns)
		summary.TotalInputTokens += sess.InputTokens
		summary.TotalOutputTokens += sess.OutputTokens
		summary.TotalCacheTokens += sess.CacheReadTokens + sess.CacheCreationTokens
		summary.TotalCostUSD += sess.NetCostUSD

		p.sink.OnSession(sess)
		for i := range sess.Turns {
			p.sink.OnTurn(&sess.Turns[i], sess)
		}

		if !dryRun {
			sessionsToSend = append(sessionsToSend, *sess)
		}
	}

	summary.Duration = time.Since(startTime)

	if !dryRun && len(sessionsToSend) > 0 {
		resp, err := p.client.SendBatch(ctx, sessionsToSend)
		if err != nil {
			summary.Errors = append(summary.Errors, fmt.Sprintf("Hub transmission failed: %v", err))
		} else if resp != nil {
			summary.AcceptedSessions = resp.AcceptedSessions
			summary.AcceptedTurns = resp.AcceptedTurns
			if len(resp.Errors) > 0 {
				summary.Errors = append(summary.Errors, resp.Errors...)
			}
		}
	}

	return summary, nil
}

// SendSession transmits an individual session to the Hub.
func (p *Pipeline) SendSession(ctx context.Context, session *models.Session) (*models.IngestionResponse, error) {
	if session == nil {
		return nil, fmt.Errorf("session is nil")
	}
	session.MachineID = p.cfg.MachineID
	return p.client.SendBatch(ctx, []models.Session{*session})
}

// SendFile parses, costs, and sends a single transcript file to the Hub.
func (p *Pipeline) SendFile(ctx context.Context, filePath string, agentOverride, projectOverride string) (*models.IngestionResponse, error) {
	sess, err := p.scannerEngine.ScanFile(ctx, filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to scan file: %w", err)
	}
	if sess == nil {
		return nil, fmt.Errorf("no matching parser or empty session for file: %s", filePath)
	}

	if agentOverride != "" {
		sess.AgentName = agentOverride
	}
	if projectOverride != "" {
		sess.ProjectName = projectOverride
	}
	sess.MachineID = p.cfg.MachineID

	return p.SendSession(ctx, sess)
}

// SendSynthetic generates and transmits a realistic test session to verify Hub connectivity and ingestion.
func (p *Pipeline) SendSynthetic(ctx context.Context, agentName, projectName, modelName string) (*models.IngestionResponse, error) {
	if agentName == "" {
		agentName = "claude_code"
	}
	if projectName == "" {
		projectName = "test-project"
	}
	if modelName == "" {
		modelName = "claude-3-7-sonnet"
	}

	now := time.Now().UTC()
	sessID := fmt.Sprintf("synthetic_%d", now.UnixNano())
	fullID := fmt.Sprintf("%s:%s", agentName, sessID)

	sess := &models.Session{
		ID:                  fullID,
		SessionID:           sessID,
		AgentName:           agentName,
		ProjectName:         projectName,
		FilePath:            "/synthetic/test.json",
		MachineID:           p.cfg.MachineID,
		CreatedAt:           now,
		UpdatedAt:           now,
		StartTime:           now.Add(-10 * time.Minute),
		EndTime:             now,
		DurationSeconds:     600,
		ModelRaw:            modelName,
		InputTokens:         125000,
		OutputTokens:        4200,
		CacheReadTokens:     850000,
		CacheCreationTokens: 12000,
		Status:              "completed",
		GitBranch:           "main",
		Turns: []models.MessageTurn{
			{
				ID:                  fmt.Sprintf("%s:1", fullID),
				SessionID:           fullID,
				TurnIndex:           1,
				Timestamp:           now.Add(-9 * time.Minute),
				Role:                "assistant",
				ModelName:           modelName,
				InputTokens:         50000,
				OutputTokens:        1500,
				CacheReadTokens:     400000,
				CacheCreationTokens: 12000,
				ToolsInvoked:        []string{"view_file", "grep_search"},
			},
			{
				ID:              fmt.Sprintf("%s:2", fullID),
				SessionID:       fullID,
				TurnIndex:       2,
				Timestamp:       now.Add(-3 * time.Minute),
				Role:            "assistant",
				ModelName:       modelName,
				InputTokens:     75000,
				OutputTokens:    2700,
				CacheReadTokens: 450000,
				ToolsInvoked:    []string{"replace_file_content", "run_command"},
			},
		},
	}

	p.pricingEngine.CostSession(ctx, sess, "", "", nil)
	return p.SendSession(ctx, sess)
}

// PingHub checks connectivity to the configured Hub endpoint.
func (p *Pipeline) PingHub(ctx context.Context) (*HubHealth, error) {
	hubURL := p.cfg.HubURL
	if hubURL == "" {
		return &HubHealth{
			Status: "OFFLINE",
			Error:  "Hub URL not configured",
		}, nil
	}

	start := time.Now()
	healthEndpoint := hubURL + "/healthz"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, healthEndpoint, nil)
	if err != nil {
		return &HubHealth{
			Status: "OFFLINE",
			HubURL: hubURL,
			Error:  err.Error(),
		}, nil
	}

	if p.cfg.AuthToken != "" {
		req.Header.Set("Authorization", "Bearer "+p.cfg.AuthToken)
	}

	client := &http.Client{Timeout: 3 * time.Second}
	resp, err := client.Do(req)
	latency := time.Since(start)

	if err != nil {
		return &HubHealth{
			Status:  "OFFLINE",
			HubURL:  hubURL,
			Latency: latency,
			Error:   err.Error(),
		}, nil
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return &HubHealth{
			Status:  "ERROR",
			HubURL:  hubURL,
			Latency: latency,
			Error:   fmt.Sprintf("received HTTP %d", resp.StatusCode),
		}, nil
	}

	body, _ := io.ReadAll(resp.Body)
	var healthData struct {
		Status  string `json:"status"`
		Version string `json:"version"`
	}
	_ = json.Unmarshal(body, &healthData)

	ver := healthData.Version
	if ver == "" {
		ver = "v2.0.0"
	}

	return &HubHealth{
		Status:        "ONLINE",
		HubURL:        hubURL,
		Latency:       latency,
		ServerVersion: ver,
	}, nil
}

// CollectSessions scans roots, filters by harness (if specified), sorts newest first, and returns up to limit sessions.
func (p *Pipeline) CollectSessions(ctx context.Context, roots []string, harnessFilter string, limit int) ([]*models.Session, error) {
	if len(roots) == 0 {
		roots = p.cfg.ScanRoots
	}

	type fileCandidate struct {
		path    string
		modTime time.Time
	}

	var candidates []fileCandidate
	registry := p.scannerEngine.GetRegistry()

	for _, root := range roots {
		fi, err := os.Stat(root)
		if err != nil {
			continue
		}

		if !fi.IsDir() {
			parser := registry.Detect(root)
			if parser != nil {
				if harnessFilter == "" || strings.EqualFold(parser.AgentName(), harnessFilter) || strings.Contains(strings.ToLower(parser.AgentName()), strings.ToLower(harnessFilter)) {
					candidates = append(candidates, fileCandidate{path: root, modTime: fi.ModTime()})
				}
			}
			continue
		}

		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				name := d.Name()
				if name == ".git" || name == "node_modules" || name == "dist" || name == ".cache" {
					return filepath.SkipDir
				}
				return nil
			}
			parser := registry.Detect(path)
			if parser != nil {
				if harnessFilter == "" || strings.EqualFold(parser.AgentName(), harnessFilter) || strings.Contains(strings.ToLower(parser.AgentName()), strings.ToLower(harnessFilter)) {
					if info, err := d.Info(); err == nil {
						candidates = append(candidates, fileCandidate{path: path, modTime: info.ModTime()})
					} else {
						candidates = append(candidates, fileCandidate{path: path, modTime: time.Now()})
					}
				}
			}
			return nil
		})
	}

	// Sort candidates by modTime descending
	sort.Slice(candidates, func(i, j int) bool {
		return candidates[i].modTime.After(candidates[j].modTime)
	})

	var sessions []*models.Session
	for _, c := range candidates {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		default:
		}

		sess, err := p.scannerEngine.ScanFile(ctx, c.path)
		if err != nil || sess == nil {
			continue
		}

		if harnessFilter != "" && !strings.EqualFold(sess.AgentName, harnessFilter) && !strings.Contains(strings.ToLower(sess.AgentName), strings.ToLower(harnessFilter)) {
			continue
		}

		sessions = append(sessions, sess)
	}

	// Sort sessions by StartTime or EndTime descending
	sort.Slice(sessions, func(i, j int) bool {
		t1 := sessions[i].StartTime
		if t1.IsZero() {
			t1 = sessions[i].CreatedAt
		}
		t2 := sessions[j].StartTime
		if t2.IsZero() {
			t2 = sessions[j].CreatedAt
		}
		return t1.After(t2)
	})

	if limit > 0 && len(sessions) > limit {
		sessions = sessions[:limit]
	}

	return sessions, nil
}

