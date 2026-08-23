package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/mattn/go-isatty"
	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/robin-paul/tokentelemetry-go/internal/tui"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var (
		daemonFlag            bool
		debounceFlag          time.Duration
		reconcileIntervalFlag time.Duration
	)

	cmd := &cobra.Command{
		Use:   "watch [paths...]",
		Short: "Monitor agent transcript directories for live token telemetry",
		Long: `Watch starts live filesystem monitoring via fsnotify across configured or specified
transcript directories. Changes are debounced, parsed, priced offline, and streamed in batches
to the TokenTelemetry Hub.

When running in an interactive terminal, it presents a rich Bubble Tea TUI dashboard.
In daemon mode (--daemon) or non-interactive shells (CI/background), it runs as a headless structured slog logger.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWatch(cmd.Context(), activeConfig, args, daemonFlag, debounceFlag, reconcileIntervalFlag)
		},
	}

	cmd.Flags().BoolVarP(&daemonFlag, "daemon", "d", false, "Run in headless background daemon mode (structured slog output)")
	cmd.Flags().DurationVar(&debounceFlag, "debounce", 250*time.Millisecond, "Filesystem event debounce duration")
	cmd.Flags().DurationVar(&reconcileIntervalFlag, "reconcile-interval", 60*time.Second, "Fallback periodic reconciliation interval")

	return cmd
}

func runWatch(ctx context.Context, cfg *collector.Config, paths []string, daemon bool, debounce, reconcileInterval time.Duration) error {
	if cfg == nil {
		cfg = collector.DefaultConfig()
	}

	if len(paths) > 0 {
		cfg.ScanRoots = paths
	}

	// Determine presentation sink
	isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
	runHeadless := daemon || cfg.Daemon || !isTerminal

	if runHeadless {
		level := parseSlogLevel(cfg.LogLevel)
		sink := collector.NewSlogSink(os.Stdout, level, false)
		pipeline, err := collector.NewPipeline(cfg, sink)
		if err != nil {
			return fmt.Errorf("failed to create collector pipeline: %w", err)
		}

		if err := pipeline.Start(ctx); err != nil {
			return fmt.Errorf("failed to start collector: %w", err)
		}

		// Block until context cancellation (SIGINT/SIGTERM)
		<-ctx.Done()

		// Graceful shutdown
		return pipeline.Stop()
	}

	// Interactive Bubble Tea TUI mode
	tuiSink := tui.NewTUISink()
	pipeline, err := collector.NewPipeline(cfg, tuiSink)
	if err != nil {
		return fmt.Errorf("failed to create collector pipeline: %w", err)
	}

	return tui.Run(ctx, cfg, pipeline, tuiSink)
}

func parseSlogLevel(levelStr string) slog.Level {
	switch strings.ToLower(strings.TrimSpace(levelStr)) {
	case "debug":
		return slog.LevelDebug
	case "warn", "warning":
		return slog.LevelWarn
	case "error":
		return slog.LevelError
	default:
		return slog.LevelInfo
	}
}
