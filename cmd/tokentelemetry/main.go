package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/robin-paul/tokentelemetry-go/internal/web"
)

var (
	Version = "1.0.0"
	Commit  = "unknown"
)

func main() {
	port := flag.Int("port", 8000, "HTTP server listening port")
	dbPath := flag.String("db", "tokentelemetry.db", "SQLite database file path")
	versionFlag := flag.Bool("version", false, "Print version information and exit")
	flag.Parse()

	if *versionFlag {
		fmt.Printf("tokentelemetry version %s (commit %s)\n", Version, Commit)
		os.Exit(0)
	}

	log.Printf("Starting TokenTelemetry Go v%s (commit: %s)", Version, Commit)
	log.Printf("Database path: %s", *dbPath)

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintf(w, `{"status":"ok","version":"%s"}`, Version)
	})
	mux.Handle("/", web.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", *port),
		Handler: mux,
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
}
