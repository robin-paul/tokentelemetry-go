package main

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/spf13/cobra"
)

func newScanCmd() *cobra.Command {
	var (
		dryRunFlag bool
		jsonFlag   bool
	)

	cmd := &cobra.Command{
		Use:   "scan [paths...]",
		Short: "Perform a one-off discovery sweep of transcript directories",
		Long: `Scan traverses the configured or specified root directories, detects and parses all
agent session files, costs them using the offline pricing catalog, and transmits them as an
ingestion batch to the TokenTelemetry Hub.

Use --dry-run to inspect discovered files and calculated token costs without sending data.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cfg := activeConfig
			if cfg == nil {
				cfg = collector.DefaultConfig()
			}

			roots := args
			if len(roots) == 0 {
				roots = cfg.ScanRoots
			}

			sink := collector.NewSilentSink()
			pipeline, err := collector.NewPipeline(cfg, sink)
			if err != nil {
				return fmt.Errorf("failed to create collector pipeline: %w", err)
			}

			summary, err := pipeline.ScanOnce(cmd.Context(), roots, dryRunFlag)
			if err != nil {
				return fmt.Errorf("scan failed: %w", err)
			}

			if jsonFlag {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(summary)
			}

			// Render formatted CLI summary
			fmt.Fprintln(out, "================================================================================")
			fmt.Fprintln(out, "⚡ TokenTelemetry Scan Sweep Results")
			fmt.Fprintln(out, "================================================================================")
			fmt.Fprintf(out, "Files Discovered:    %d\n", summary.TotalFiles)
			fmt.Fprintf(out, "Sessions Parsed:     %d\n", summary.ParsedSessions)
			fmt.Fprintf(out, "Message Turns:       %d\n", summary.TotalTurns)
			fmt.Fprintf(out, "Input Tokens:        %s\n", humanize.Comma(summary.TotalInputTokens))
			fmt.Fprintf(out, "Output Tokens:       %s\n", humanize.Comma(summary.TotalOutputTokens))
			fmt.Fprintf(out, "Cache Read/Write:    %s\n", humanize.Comma(summary.TotalCacheTokens))
			fmt.Fprintf(out, "Estimated Cost:      $%.4f USD\n", summary.TotalCostUSD)
			fmt.Fprintf(out, "Scan Duration:       %v\n", summary.Duration.Round(100*time.Microsecond))
			fmt.Fprintln(out, "--------------------------------------------------------------------------------")

			if dryRunFlag {
				fmt.Fprintln(out, "Mode:                DRY RUN (No batches sent to Hub)")
			} else {
				fmt.Fprintf(out, "Hub Target:          %s\n", cfg.HubURL)
				fmt.Fprintf(out, "Accepted Sessions:   %d\n", summary.AcceptedSessions)
				fmt.Fprintf(out, "Accepted Turns:      %d\n", summary.AcceptedTurns)
				if len(summary.Errors) > 0 {
					fmt.Fprintln(out, "\nErrors / Warnings:")
					for _, e := range summary.Errors {
						fmt.Fprintf(out, "  • %s\n", e)
					}
				}
			}
			fmt.Fprintln(out, "================================================================================")

			return nil
		},
	}

	cmd.Flags().BoolVar(&dryRunFlag, "dry-run", false, "Scan and calculate costs without transmitting to Hub")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output scan summary formatted as JSON")

	return cmd
}
