package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/api"
	"github.com/robin-paul/tokentelemetry-go/internal/events"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/robin-paul/tokentelemetry-go/internal/pricing"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
	"github.com/robin-paul/tokentelemetry-go/internal/store"
	"github.com/robin-paul/tokentelemetry-go/internal/watcher"
	"github.com/robin-paul/tokentelemetry-go/internal/web"
)

var (
	Version = "1.0.0"
	Commit  = "unknown"
)

func main() {
	port := flag.Int("port", 8000, "HTTP server listening port")
	dbPath := flag.String("db", "tokentelemetry.db", "SQLite database file path")
	authToken := flag.String("auth-token", os.Getenv("TT_AUTH_TOKEN"), "Bearer token for non-loopback clients")
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	noWatch := flag.Bool("no-watch", false, "Disable live filesystem watcher and background scanner")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("tokentelemetry version %s (commit %s)\n", Version, Commit)
		os.Exit(0)
	}

	log.Printf("Starting TokenTelemetry Go v%s (commit: %s)", Version, Commit)
	log.Printf("Database path: %s", *dbPath)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 1. Initialize SQLite Store & Migrations
	db, err := store.Open(*dbPath)
	if err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer db.Close()

	if err := db.Migrate(ctx); err != nil {
		log.Fatalf("Failed to run schema migrations: %v", err)
	}

	// 2. Initialize Pricing Engine
	pricingEngine, err := pricing.NewEngine()
	if err != nil {
		log.Fatalf("Failed to load pricing engine: %v", err)
	}

	// 3. Initialize SSE Broker
	broker := events.NewBroker(15 * time.Second)
	go broker.Start(ctx)
	defer broker.Stop()

	// 4. Initialize Scanner Engine
	scannerEngine := scanner.NewEngine(db, pricingEngine, scanner.Config{
		OnSessionEvent: func(eventType events.EventType, session *models.Session) {
			_ = broker.Broadcast(events.EventPayload{
				Type: eventType,
				Data: events.SessionEventData{Session: *session},
			})
		},
		OnScanProgress: func(currentFile string, processed, total int) {
			_ = broker.Broadcast(events.EventPayload{
				Type: events.EventScanProgress,
				Data: events.ScanProgressData{
					TotalFiles:     total,
					ProcessedFiles: processed,
					CurrentFile:    currentFile,
					IsComplete:     processed >= total,
				},
			})
		},
	})
	scannerEngine.Start(ctx)
	defer scannerEngine.Stop()

	// 5. Initialize Watcher & Reconciler if enabled
	if !*noWatch {
		roots := scanner.DiscoverDefaultRoots()
		log.Printf("Discovered %d agent transcript roots to monitor", len(roots))

		w, err := watcher.NewWatcher(scannerEngine, watcher.Config{
			DebounceDuration: 250 * time.Millisecond,
		})
		if err == nil {
			for _, r := range roots {
				_ = w.AddRoot(r)
			}
			w.Start(ctx)
			defer func() { _ = w.Stop() }()
		}

		reconciler := watcher.NewReconciler(scannerEngine, watcher.ReconcilerConfig{
			Interval: 60 * time.Second,
			Roots:    roots,
		})
		reconciler.Start(ctx)
		defer reconciler.Stop()

		// Trigger initial sweep in background
		go func() {
			_ = reconciler.Sweep(context.Background())
		}()
	}

	// 6. Build API Router with chi and Static Web Assets
	apiServer := api.NewServer(db, pricingEngine, scannerEngine, broker, api.Config{
		AuthToken:  *authToken,
		Version:    Version,
		Commit:     Commit,
		WebHandler: web.Handler(),
		Logger:     log.Printf,
	})

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: apiServer.Router(),
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	go func() {
		log.Printf("Listening on http://localhost:%d", *port)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("HTTP server failure: %v", err)
		}
	}()

	<-sigChan
	log.Println("Shutting down gracefully...")

	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("Server shutdown error: %v", err)
	}
}
