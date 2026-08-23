package main

import (
	"fmt"

	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/robin-paul/tokentelemetry-go/internal/models"
	"github.com/spf13/cobra"
)

func newSendCmd() *cobra.Command {
	var (
		filePathFlag  string
		agentFlag     string
		projectFlag   string
		modelFlag     string
		syntheticFlag bool
	)

	cmd := &cobra.Command{
		Use:   "send",
		Short: "Inject a synthetic or real transcript file for verification testing",
		Long: `Send transmits an agent transcript file or synthetic test session directly to the
TokenTelemetry Hub to verify end-to-end network connectivity, Bearer token authentication,
and SQLite persistence.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			out := cmd.OutOrStdout()
			cfg := activeConfig
			if cfg == nil {
				cfg = collector.DefaultConfig()
			}

			sink := collector.NewConsoleSink(out)
			pipeline, err := collector.NewPipeline(cfg, sink)
			if err != nil {
				return fmt.Errorf("failed to create collector pipeline: %w", err)
			}

			ctx := cmd.Context()
			var resp *models.IngestionResponse

			if filePathFlag != "" {
				fmt.Fprintf(out, "Parsing and sending transcript file: %s (agent override: %q, project override: %q)...\n",
					filePathFlag, agentFlag, projectFlag)
				resp, err = pipeline.SendFile(ctx, filePathFlag, agentFlag, projectFlag)
			} else {
				// Default to synthetic generation
				syntheticAgent := agentFlag
				if syntheticAgent == "" {
					syntheticAgent = "claude_code"
				}
				syntheticProject := projectFlag
				if syntheticProject == "" {
					syntheticProject = "synthetic-verification"
				}
				syntheticModel := modelFlag
				if syntheticModel == "" {
					syntheticModel = "claude-3-7-sonnet"
				}

				fmt.Fprintf(out, "Generating synthetic test session for agent %q (project: %q, model: %q)...\n",
					syntheticAgent, syntheticProject, syntheticModel)
				resp, err = pipeline.SendSynthetic(ctx, syntheticAgent, syntheticProject, syntheticModel)
			}

			if err != nil {
				return fmt.Errorf("transmission failed: %w", err)
			}

			fmt.Fprintln(out, "\n================================================================================")
			fmt.Fprintln(out, "⚡ TokenTelemetry Ingestion Result")
			fmt.Fprintln(out, "================================================================================")
			fmt.Fprintf(out, "Status:             %s\n", resp.Status)
			fmt.Fprintf(out, "Batch ID:           %s\n", resp.BatchID)
			fmt.Fprintf(out, "Accepted Sessions:  %d\n", resp.AcceptedSessions)
			fmt.Fprintf(out, "Accepted Turns:     %d\n", resp.AcceptedTurns)
			fmt.Fprintf(out, "Rejected Sessions:  %d\n", resp.RejectedSessions)
			fmt.Fprintf(out, "Server Time:        %s\n", resp.ServerTime.Format("2006-01-02 15:04:05 UTC"))

			if len(resp.Errors) > 0 {
				fmt.Fprintln(out, "\nErrors returned by server:")
				for _, e := range resp.Errors {
					fmt.Fprintf(out, "  • %s\n", e)
				}
			}
			fmt.Fprintln(out, "================================================================================")

			return nil
		},
	}

	cmd.Flags().StringVarP(&filePathFlag, "file", "f", "", "Path to transcript file to parse and inject")
	cmd.Flags().StringVarP(&agentFlag, "agent", "a", "", "Agent name override")
	cmd.Flags().StringVarP(&projectFlag, "project", "p", "", "Project name override")
	cmd.Flags().StringVarP(&modelFlag, "model", "m", "", "Model name override for synthetic sessions")
	cmd.Flags().BoolVar(&syntheticFlag, "synthetic", false, "Generate and transmit a synthetic test session")

	return cmd
}
