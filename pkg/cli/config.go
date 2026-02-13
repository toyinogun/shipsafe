// Package cli provides CLI-specific logic including configuration loading.
package cli

import (
	"fmt"
	"log/slog"
	"os"

	"gopkg.in/yaml.v3"
)

// Config represents the .shipsafe.yml configuration file.
type Config struct {
	Version    string          `yaml:"version"`
	Thresholds ThresholdConfig `yaml:"thresholds"`
	Analyzers  AnalyzersConfig `yaml:"analyzers"`
	AI         AIConfig        `yaml:"ai"`
	Output     OutputConfig    `yaml:"output"`
	CI         CIConfig        `yaml:"ci"`
}

// AIConfig holds configuration for the AI-powered code review module.
type AIConfig struct {
	Enabled   bool   `yaml:"enabled"`
	Provider  string `yaml:"provider"`
	Endpoint  string `yaml:"endpoint"`
	Model     string `yaml:"model"`
	APIKeyEnv string `yaml:"api_key_env"`
}

// ThresholdConfig holds the trust score thresholds.
type ThresholdConfig struct {
	Green  int `yaml:"green"`
	Yellow int `yaml:"yellow"`
}

// AnalyzersConfig holds per-analyzer configuration.
type AnalyzersConfig struct {
	Complexity AnalyzerModuleConfig `yaml:"complexity"`
	Coverage   AnalyzerModuleConfig `yaml:"coverage"`
	Secrets    AnalyzerModuleConfig `yaml:"secrets"`
	Imports    AnalyzerModuleConfig `yaml:"imports"`
	Patterns   AnalyzerModuleConfig `yaml:"patterns"`
}

// AnalyzerModuleConfig configures a single analyzer module.
type AnalyzerModuleConfig struct {
	Enabled   *bool `yaml:"enabled"`
	Threshold int   `yaml:"threshold,omitempty"`
}

// IsEnabled reports whether this analyzer module is enabled.
// Returns true by default if not explicitly set.
func (a AnalyzerModuleConfig) IsEnabled() bool {
	if a.Enabled == nil {
		return true
	}
	return *a.Enabled
}

// OutputConfig controls report output settings.
type OutputConfig struct {
	Format  string `yaml:"format"`
	Verbose bool   `yaml:"verbose"`
}

// CIConfig controls CI integration behavior.
type CIConfig struct {
	FailOn        string `yaml:"fail_on"`
	Comment       bool   `yaml:"comment"`
	CommentFormat string `yaml:"comment_format"`
}

// LoadConfig reads and parses a .shipsafe.yml configuration file.
// If path is empty, it looks for .shipsafe.yml in the current directory.
// If the default config file is not found, sensible defaults are returned.
// If an explicitly specified config file is not found, an error is returned.
func LoadConfig(path string) (*Config, error) {
	useDefault := path == ""
	if useDefault {
		path = ".shipsafe.yml"
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) && useDefault {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("cli: reading config %s: %w", path, err)
	}

	cfg := &Config{}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("cli: parsing config %s: %w", path, err)
	}

	applyDefaults(cfg)
	return cfg, nil
}

// DefaultConfig returns a Config with sensible defaults matching the documented
// .shipsafe.yml schema.
func DefaultConfig() *Config {
	cfg := &Config{Version: "1"}
	applyDefaults(cfg)
	return cfg
}

// applyDefaults fills in zero-value fields with sensible defaults.
func applyDefaults(cfg *Config) {
	if cfg.Thresholds.Green == 0 {
		cfg.Thresholds.Green = 80
	}
	if cfg.Thresholds.Yellow == 0 {
		cfg.Thresholds.Yellow = 50
	}
	if cfg.Analyzers.Complexity.Threshold == 0 {
		cfg.Analyzers.Complexity.Threshold = 15
	}
	if cfg.Output.Format == "" {
		cfg.Output.Format = "terminal"
	}
	if cfg.CI.FailOn == "" {
		cfg.CI.FailOn = "red"
	}
	if cfg.CI.CommentFormat == "" {
		cfg.CI.CommentFormat = "markdown"
	}
	if cfg.AI.APIKeyEnv == "" {
		cfg.AI.APIKeyEnv = "SHIPSAFE_AI_API_KEY"
	}

	// Auto-enable AI review when API key is present in environment.
	if !cfg.AI.Enabled && os.Getenv(cfg.AI.APIKeyEnv) != "" {
		cfg.AI.Enabled = true
		slog.Info("AI review auto-enabled (API key detected)")
	}
}
