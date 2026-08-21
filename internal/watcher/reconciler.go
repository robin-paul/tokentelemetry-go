package watcher

import (
	"context"
	"sync"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
)

// Reconciler periodically sweeps log roots as a safety net against dropped filesystem events.
type Reconciler struct {
	scanner  *scanner.Engine
	interval time.Duration
	roots    []string
	running  bool
	runMu    sync.Mutex
	cancel   context.CancelFunc
	wg       sync.WaitGroup
}

// ReconcilerConfig provides configuration options for the fallback reconciler.
type ReconcilerConfig struct {
	Interval time.Duration
	Roots    []string
}

// NewReconciler creates a new Reconciler instance.
func NewReconciler(eng *scanner.Engine, cfg ReconcilerConfig) *Reconciler {
	interval := cfg.Interval
	if interval <= 0 {
		interval = 60 * time.Second
	}

	roots := cfg.Roots
	if len(roots) == 0 {
		roots = scanner.DiscoverDefaultRoots()
	}

	return &Reconciler{
		scanner:  eng,
		interval: interval,
		roots:    roots,
	}
}

// Start begins the periodic reconciler ticker in the background.
func (r *Reconciler) Start(ctx context.Context) {
	r.runMu.Lock()
	if r.running {
		r.runMu.Unlock()
		return
	}
	r.running = true

	recCtx, cancel := context.WithCancel(ctx)
	r.cancel = cancel
	r.runMu.Unlock()

	r.wg.Add(1)
	go func() {
		defer r.wg.Done()
		r.loop(recCtx)
	}()
}

// Stop terminates the reconciler ticker.
func (r *Reconciler) Stop() {
	r.runMu.Lock()
	if !r.running {
		r.runMu.Unlock()
		return
	}
	r.running = false
	if r.cancel != nil {
		r.cancel()
	}
	r.runMu.Unlock()

	r.wg.Wait()
}

// Sweep runs an immediate reconciliation pass across all roots.
func (r *Reconciler) Sweep(ctx context.Context) error {
	if r.scanner == nil {
		return nil
	}
	return r.scanner.ScanRoots(ctx, r.roots)
}

func (r *Reconciler) loop(ctx context.Context) {
	ticker := time.NewTicker(r.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			_ = r.Sweep(ctx)
		}
	}
}
