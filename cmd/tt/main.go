package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/spf13/cobra"
)

var (
	// Version and Commit populated at link time
	Version = "2.0.0"
	Commit  = "dev"

	configFileFlag string
	hubURLFlag     string
	authTokenFlag  string
	machineIDFlag  string
	logLevelFlag   string

	activeConfig *collector.Config
)

// NewRootCmd constructs the root Cobra command for the TokenTelemetry collector CLI.
func NewRootCmd() *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "tt",
		Short: "TokenTelemetry Developer Telemetry Collector",
		Long: `TokenTelemetry Collector (tt) is a lightweight CLI utility that passively monitors
local AI coding agent transcripts, parses token usage, calculates financial costs offline,
and streams telemetry batches to the TokenTelemetry Hub.`,
		Version: fmt.Sprintf("%s (%s)", Version, Commit),
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := collector.LoadConfig(configFileFlag)
			if err != nil {
				return err
			}

			// Apply CLI flag overrides (highest precedence)
			if hubURLFlag != "" {
				cfg.HubURL = hubURLFlag
			}
			if authTokenFlag != "" {
				cfg.AuthToken = authTokenFlag
			}
			if machineIDFlag != "" {
				cfg.MachineID = machineIDFlag
			}
			if logLevelFlag != "" {
				cfg.LogLevel = logLevelFlag
			}

			activeConfig = cfg
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action when invoked with no subcommand is `tt watch`
			return runWatch(cmd.Context(), activeConfig, nil, false, 0, 0)
		},
	}

	// Persistent Global Flags
	rootCmd.PersistentFlags().StringVarP(&configFileFlag, "config", "c", "", "Path to configuration YAML file (default ~/.tokentelemetry/config.yaml)")
	rootCmd.PersistentFlags().StringVar(&hubURLFlag, "hub", "", "Hub endpoint URL (e.g. http://localhost:8000)")
	rootCmd.PersistentFlags().StringVar(&authTokenFlag, "api-key", "", "Hub Bearer authentication token")
	rootCmd.PersistentFlags().StringVar(&authTokenFlag, "auth-token", "", "Hub Bearer authentication token (alias for --api-key)")
	_ = rootCmd.PersistentFlags().MarkHidden("auth-token")
	rootCmd.PersistentFlags().StringVar(&machineIDFlag, "machine-id", "", "Collector machine identifier")
	rootCmd.PersistentFlags().StringVar(&logLevelFlag, "log-level", "", "Logging level (debug, info, warn, error)")

	// Subcommands
	rootCmd.AddCommand(newWatchCmd())
	rootCmd.AddCommand(newScanCmd())
	rootCmd.AddCommand(newConfigCmd())
	rootCmd.AddCommand(newStatusCmd())
	rootCmd.AddCommand(newSendCmd())

	return rootCmd
}

func main() {
	ctx, cancel := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer cancel()

	rootCmd := NewRootCmd()
	if err := rootCmd.ExecuteContext(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
