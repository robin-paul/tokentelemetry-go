package main

import (
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/mattn/go-isatty"
	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/robin-paul/tokentelemetry-go/internal/tui"
	"github.com/spf13/cobra"
)

func newSessionsCmd() *cobra.Command {
	var (
		harnessFlag string
		limitFlag   int
		plainFlag   bool
		jsonFlag    bool
	)

	cmd := &cobra.Command{
		Use:     "sessions [paths...]",
		Aliases: []string{"sess", "debug"},
		Short:   "Browse and debug discovered agent sessions",
		Long: `Sessions inspects transcript directories, finds the latest X agent execution sessions,
extracts turn-by-turn debugging telemetry, and presents them in an interactive TUI browser
or formatted static table.

Use --harness to filter by agent harness (e.g., antigravity, claude_code, cursor, opencode).
Use --limit to control how many recent sessions to retrieve (default: 10).
Use --plain for non-interactive static terminal output or --json for scripting.`,
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

			sessions, err := pipeline.CollectSessions(cmd.Context(), roots, harnessFlag, limitFlag)
			if err != nil {
				return fmt.Errorf("failed to collect sessions: %w", err)
			}

			if jsonFlag {
				enc := json.NewEncoder(out)
				enc.SetIndent("", "  ")
				return enc.Encode(sessions)
			}

			isTerminal := isatty.IsTerminal(os.Stdout.Fd()) || isatty.IsCygwinTerminal(os.Stdout.Fd())
			if plainFlag || !isTerminal {
				fmt.Fprintln(out, "================================================================================")
				fmt.Fprintln(out, "⚡ TokenTelemetry Discovered Sessions")
				if harnessFlag != "" {
					fmt.Fprintf(out, "Filter:              Harness=%s\n", harnessFlag)
				}
				fmt.Fprintf(out, "Sessions Found:      %d (Limit: %d)\n", len(sessions), limitFlag)
				fmt.Fprintln(out, "================================================================================")

				if len(sessions) == 0 {
					fmt.Fprintln(out, "No matching sessions found in configured scan roots.")
					fmt.Fprintln(out, "================================================================================")
					return nil
				}

				for i, s := range sessions {
					ts := s.StartTime
					if ts.IsZero() {
						ts = s.CreatedAt
					}
					durStr := fmt.Sprintf("%.1fs", s.DurationSeconds)
					if s.DurationSeconds <= 0 && !s.StartTime.IsZero() && !s.EndTime.IsZero() {
						durStr = fmt.Sprintf("%.1fs", s.EndTime.Sub(s.StartTime).Seconds())
					}

					modelStr := s.ModelRaw
					if s.ModelResolved != "" && s.ModelResolved != s.ModelRaw {
						modelStr = fmt.Sprintf("%s (resolved: %s)", s.ModelRaw, s.ModelResolved)
					}

					fmt.Fprintf(out, "\n[%d] Session ID:     %s\n", i+1, s.ID)
					fmt.Fprintf(out, "    Harness:        %s\n", s.AgentName)
					fmt.Fprintf(out, "    Model:          %s\n", modelStr)
					fmt.Fprintf(out, "    File:           %s\n", s.FilePath)
					fmt.Fprintf(out, "    Timestamp:      %s (Duration: %s)\n", ts.Format(time.RFC3339), durStr)
					fmt.Fprintf(out, "    Turns:          %d\n", len(s.Turns))
					fmt.Fprintf(out, "    Tokens (In/Out):%s / %s (Cache Read: %s)\n",
						humanize.Comma(s.InputTokens),
						humanize.Comma(s.OutputTokens),
						humanize.Comma(s.CacheReadTokens),
					)
					fmt.Fprintf(out, "    Cost (USD):     $%.4f Net (Gross: $%.4f)\n", s.NetCostUSD, s.GrossCostUSD)

					if len(s.Turns) > 0 {
						fmt.Fprintln(out, "    Turn Samples:")
						maxTurns := 3
						if len(s.Turns) < maxTurns {
							maxTurns = len(s.Turns)
						}
						for _, t := range s.Turns[len(s.Turns)-maxTurns:] {
							toolsStr := ""
							if len(t.ToolsInvoked) > 0 {
								toolsStr = fmt.Sprintf(" tools=%v", t.ToolsInvoked)
							}
							fmt.Fprintf(out, "      • Turn #%-2d [%-9s] [%-16s] in:%-5s out:%-4s cost:$%.4f%s\n",
								t.TurnIndex, t.Role, t.ModelName,
								humanize.Comma(t.InputTokens), humanize.Comma(t.OutputTokens),
								t.CostUSD, toolsStr,
							)
						}
					}
				}
				fmt.Fprintln(out, "\n================================================================================")
				return nil
			}

			// Launch Interactive TUI Sessions Browser
			return tui.RunSessionsBrowser(cmd.Context(), cfg, pipeline, sessions, harnessFlag)
		},
	}

	cmd.Flags().StringVarP(&harnessFlag, "harness", "a", "", "Filter sessions by agent harness (e.g. antigravity, claude_code, cursor, opencode)")
	cmd.Flags().StringVar(&harnessFlag, "agent", "", "Alias for --harness")
	_ = cmd.Flags().MarkHidden("agent")
	cmd.Flags().IntVarP(&limitFlag, "limit", "n", 10, "Maximum number of recent sessions to display")
	cmd.Flags().BoolVar(&plainFlag, "plain", false, "Output formatted static terminal table instead of full-screen TUI")
	cmd.Flags().BoolVar(&jsonFlag, "json", false, "Output session debugging data formatted as JSON")

	return cmd
}
