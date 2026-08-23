package main

import (
	"fmt"
	"io"

	"github.com/robin-paul/tokentelemetry-go/internal/collector"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Inspect and manage local collector configuration",
		Long: `Config allows viewing and editing the persistent local collector configuration
stored in ~/.tokentelemetry/config.yaml.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Default action when running `tt config` is to list all settings
			return runConfigList(cmd.OutOrStdout())
		},
	}

	cmd.AddCommand(newConfigGetCmd())
	cmd.AddCommand(newConfigSetCmd())
	cmd.AddCommand(newConfigListCmd())
	cmd.AddCommand(newConfigPathCmd())

	return cmd
}

func newConfigGetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: "Get the value of a configuration setting",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := activeConfig
			if cfg == nil {
				cfg = collector.DefaultConfig()
			}
			val, err := collector.GetConfigValue(cfg, args[0])
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), val)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set the value of a configuration setting and persist to file",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := activeConfig
			if cfg == nil {
				cfg = collector.DefaultConfig()
			}

			key := args[0]
			val := args[1]

			if err := collector.SetConfigValue(cfg, key, val); err != nil {
				return err
			}

			targetPath := configFileFlag
			if targetPath == "" {
				defaultPath, err := collector.DefaultConfigPath()
				if err != nil {
					return err
				}
				targetPath = defaultPath
			}

			if err := collector.SaveConfig(targetPath, cfg); err != nil {
				return fmt.Errorf("failed to save config to %s: %w", targetPath, err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Updated %s = %q in %s\n", key, val, targetPath)
			return nil
		},
	}
}

func newConfigListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List all active configuration settings in YAML format",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runConfigList(cmd.OutOrStdout())
		},
	}
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Print the path to the configuration file",
		RunE: func(cmd *cobra.Command, args []string) error {
			targetPath := configFileFlag
			if targetPath == "" {
				defaultPath, err := collector.DefaultConfigPath()
				if err != nil {
					return err
				}
				targetPath = defaultPath
			}
			fmt.Fprintln(cmd.OutOrStdout(), targetPath)
			return nil
		},
	}
}

func runConfigList(out io.Writer) error {
	cfg := activeConfig
	if cfg == nil {
		cfg = collector.DefaultConfig()
	}

	targetPath := configFileFlag
	if targetPath == "" {
		if p, err := collector.DefaultConfigPath(); err == nil {
			targetPath = p
		}
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to format YAML config: %w", err)
	}

	fmt.Fprintf(out, "# TokenTelemetry Configuration (%s)\n", targetPath)
	out.Write(data)
	return nil
}
