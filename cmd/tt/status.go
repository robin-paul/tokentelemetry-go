package main

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check collector status, transcript roots, and Hub connectivity",
		Long: `Status inspects the local machine environment, checks transcript directory discovery,
verifies authentication, and pings the TokenTelemetry Hub endpoint to report connection health.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cfg := activeConfig
			if cfg == nil {
				cfg = collector.DefaultConfig()
			}

			sink := collector.NewSilentSink()
			pipeline, err := collector.NewPipeline(cfg, sink)
			if err != nil {
				return fmt.Errorf("failed to initialize collector: %w", err)
			}

			ctx := cmd.Context()
			hubHealth, err := pipeline.PingHub(ctx)
			if err != nil {
				hubHealth = &collector.HubHealth{
					Status: "OFFLINE",
					HubURL: cfg.HubURL,
					Error:  err.Error(),
				}
			}

			configPath := configFileFlag
			if configPath == "" {
				if p, err := collector.DefaultConfigPath(); err == nil {
					configPath = p
				}
			}

			configExists := "not created (using defaults)"
			if _, err := os.Stat(configPath); err == nil {
				configExists = "present"
			}

			authStatus := "Not configured"
			if cfg.AuthToken != "" {
				masked := cfg.AuthToken
				if len(masked) > 8 {
					masked = masked[:4] + "..." + masked[len(masked)-4:]
				} else {
					masked = "****"
				}
				authStatus = fmt.Sprintf("Configured (Bearer %s)", masked)
			}

			hostname, _ := os.Hostname()

			fmt.Fprintln(out, "================================================================================")
			fmt.Fprintln(out, "⚡ TokenTelemetry Collector Status")
			fmt.Fprintln(out, "================================================================================")
			fmt.Fprintf(out, "Machine ID:          %s\n", cfg.MachineID)
			fmt.Fprintf(out, "Hostname:            %s (%s/%s)\n", hostname, runtime.GOOS, runtime.GOARCH)
			fmt.Fprintf(out, "Collector Version:   %s (%s)\n", Version, Commit)
			fmt.Fprintf(out, "Configuration File:  %s [%s]\n", configPath, configExists)
			fmt.Fprintf(out, "Log Level:           %s\n", cfg.LogLevel)
			fmt.Fprintf(out, "Batch Settings:      %d items / %dms flush\n", cfg.BatchSize, cfg.FlushMS)
			fmt.Fprintln(out, "--------------------------------------------------------------------------------")
			fmt.Fprintf(out, "Hub Endpoint:        %s\n", cfg.HubURL)
			if hubHealth.Status == "ONLINE" {
				fmt.Fprintf(out, "Hub Connectivity:    🟢 ONLINE (Latency: %v, Version: %s)\n",
					hubHealth.Latency.Round(10*time.Microsecond), hubHealth.ServerVersion)
			} else {
				fmt.Fprintf(out, "Hub Connectivity:    🔴 %s (%s)\n", hubHealth.Status, hubHealth.Error)
			}
			fmt.Fprintf(out, "Authentication:      %s\n", authStatus)
			fmt.Fprintln(out, "--------------------------------------------------------------------------------")
			fmt.Fprintln(out, "Agent Transcript Discovery Roots:")

			registry := pipeline.Scanner().GetRegistry()
			if len(cfg.ScanRoots) == 0 {
				fmt.Fprintln(out, "  (No scan roots configured)")
			} else {
				for _, root := range cfg.ScanRoots {
					fi, err := os.Stat(root)
					if err != nil {
						fmt.Fprintf(out, "  ⚪ %-40s (directory does not exist)\n", shortenPath(root))
						continue
					}

					matchingCount := 0
					if fi.IsDir() {
						_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
							if err == nil && !d.IsDir() && registry.Detect(path) != nil {
								matchingCount++
							}
							return nil
						})
						fmt.Fprintf(out, "  🟢 %-40s (%d active files)\n", shortenPath(root), matchingCount)
					} else {
						if registry.Detect(root) != nil {
							matchingCount = 1
						}
						fmt.Fprintf(out, "  🟢 %-40s (single file)\n", shortenPath(root))
					}
				}
			}

			fmt.Fprintln(out, "================================================================================")
			return nil
		},
	}

	return cmd
}

func shortenPath(path string) string {
	home, err := os.UserHomeDir()
	if err == nil && strings.HasPrefix(path, home) {
		return "~" + strings.TrimPrefix(path, home)
	}
	return path
}
