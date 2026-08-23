package collector

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/robin-paul/tokentelemetry-go/internal/scanner"
	"gopkg.in/yaml.v3"
)

// Config holds client collector configurations.
type Config struct {
	HubURL       string   `yaml:"hub_url" json:"hub_url"`
	AuthToken    string   `yaml:"auth_token" json:"auth_token"`
	MachineID    string   `yaml:"machine_id" json:"machine_id"`
	ScanRoots    []string `yaml:"scan_roots" json:"scan_roots"`
	LogLevel     string   `yaml:"log_level" json:"log_level"`
	BatchSize    int      `yaml:"batch_size" json:"batch_size"`
	FlushMS      int      `yaml:"flush_ms" json:"flush_ms"`
	Daemon       bool     `yaml:"daemon" json:"daemon"`
	PowerProfile string   `yaml:"power_profile" json:"power_profile"`
	MaxRetries   int      `yaml:"max_retries" json:"max_retries"`
	TimeoutSec   int      `yaml:"timeout_sec" json:"timeout_sec"`
}

// DefaultConfig returns default collector configuration.
func DefaultConfig() *Config {
	machineID := generateMachineID()

	return &Config{
		HubURL:       "http://localhost:8000",
		AuthToken:    "",
		MachineID:    machineID,
		ScanRoots:    scanner.DiscoverDefaultRoots(),
		LogLevel:     "info",
		BatchSize:    50,
		FlushMS:      500,
		Daemon:       false,
		PowerProfile: "default",
		MaxRetries:   5,
		TimeoutSec:   15,
	}
}

// DefaultConfigPath returns the standard ~/.tokentelemetry/config.yaml path.
func DefaultConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to determine user home directory: %w", err)
	}
	return filepath.Join(home, ".tokentelemetry", "config.yaml"), nil
}

// LoadConfig loads the configuration file if present, falling back to defaults and applying environment variables.
func LoadConfig(customPath string) (*Config, error) {
	cfg := DefaultConfig()

	path := customPath
	if path == "" {
		defaultPath, err := DefaultConfigPath()
		if err == nil {
			path = defaultPath
		}
	}

	if path != "" {
		if data, err := os.ReadFile(path); err == nil {
			if err := yaml.Unmarshal(data, cfg); err != nil {
				return nil, fmt.Errorf("failed to parse YAML configuration at %s: %w", path, err)
			}
		}
	}

	ApplyEnvOverrides(cfg)
	return cfg, nil
}

// SaveConfig writes the given configuration out to a YAML file.
func SaveConfig(targetPath string, cfg *Config) error {
	path := targetPath
	if path == "" {
		defaultPath, err := DefaultConfigPath()
		if err != nil {
			return err
		}
		path = defaultPath
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory %s: %w", dir, err)
	}

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("failed to marshal YAML config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file %s: %w", path, err)
	}

	return nil
}

// ApplyEnvOverrides overrides config fields with environment variables if set.
func ApplyEnvOverrides(cfg *Config) {
	if v := os.Getenv("TT_HUB_URL"); v != "" {
		cfg.HubURL = v
	} else if v := os.Getenv("TOKEN_TELEMETRY_HUB_URL"); v != "" {
		cfg.HubURL = v
	}

	if v := os.Getenv("TT_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	} else if v := os.Getenv("TOKEN_TELEMETRY_AUTH_TOKEN"); v != "" {
		cfg.AuthToken = v
	}

	if v := os.Getenv("TT_MACHINE_ID"); v != "" {
		cfg.MachineID = v
	}

	if v := os.Getenv("TT_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}

	if v := os.Getenv("TT_SCAN_DIR"); v != "" {
		dirs := strings.Split(v, ",")
		var cleaned []string
		for _, d := range dirs {
			trimmed := strings.TrimSpace(d)
			if trimmed != "" {
				cleaned = append(cleaned, trimmed)
			}
		}
		if len(cleaned) > 0 {
			cfg.ScanRoots = cleaned
		}
	}
}

// GetConfigValue reads a string representation of a specific config property by name.
func GetConfigValue(cfg *Config, key string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hub_url", "hub", "url":
		return cfg.HubURL, nil
	case "auth_token", "token", "api_key", "apikey":
		return cfg.AuthToken, nil
	case "machine_id", "machine":
		return cfg.MachineID, nil
	case "log_level", "loglevel":
		return cfg.LogLevel, nil
	case "batch_size", "batchsize":
		return strconv.Itoa(cfg.BatchSize), nil
	case "flush_ms", "flushms":
		return strconv.Itoa(cfg.FlushMS), nil
	case "daemon":
		return strconv.FormatBool(cfg.Daemon), nil
	case "power_profile", "powerprofile":
		return cfg.PowerProfile, nil
	case "scan_roots", "roots", "scan_dirs":
		return strings.Join(cfg.ScanRoots, ", "), nil
	default:
		return "", fmt.Errorf("unknown configuration key: %q", key)
	}
}

// SetConfigValue updates a specific config property by key name.
func SetConfigValue(cfg *Config, key, value string) error {
	trimmedVal := strings.TrimSpace(value)
	switch strings.ToLower(strings.TrimSpace(key)) {
	case "hub_url", "hub", "url":
		cfg.HubURL = trimmedVal
	case "auth_token", "token", "api_key", "apikey":
		cfg.AuthToken = trimmedVal
	case "machine_id", "machine":
		cfg.MachineID = trimmedVal
	case "log_level", "loglevel":
		cfg.LogLevel = trimmedVal
	case "batch_size", "batchsize":
		n, err := strconv.Atoi(trimmedVal)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid batch_size %q: must be positive integer", value)
		}
		cfg.BatchSize = n
	case "flush_ms", "flushms":
		n, err := strconv.Atoi(trimmedVal)
		if err != nil || n <= 0 {
			return fmt.Errorf("invalid flush_ms %q: must be positive integer", value)
		}
		cfg.FlushMS = n
	case "daemon":
		b, err := strconv.ParseBool(trimmedVal)
		if err != nil {
			return fmt.Errorf("invalid daemon boolean %q: must be true or false", value)
		}
		cfg.Daemon = b
	case "power_profile", "powerprofile":
		cfg.PowerProfile = trimmedVal
	case "scan_roots", "roots", "scan_dirs":
		parts := strings.Split(trimmedVal, ",")
		var roots []string
		for _, p := range parts {
			t := strings.TrimSpace(p)
			if t != "" {
				roots = append(roots, t)
			}
		}
		cfg.ScanRoots = roots
	default:
		return fmt.Errorf("unknown configuration key: %q", key)
	}
	return nil
}

func generateMachineID() string {
	hostname, err := os.Hostname()
	if err == nil && hostname != "" {
		return fmt.Sprintf("%s-%s", hostname, uuid.New().String()[:8])
	}
	return uuid.New().String()
}
