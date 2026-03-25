// Package config loads and validates consul-review configuration.
package config

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/viper"
)

// SupportedConsuls maps consul name → binary name.
var SupportedConsuls = map[string]string{
	"gemini":  "gemini",
	"copilot": "copilot",
	"oz":      "oz",
	"claude":  "claude",
	"codex":   "codex",
}

// defaultExtraArgs holds the default non-interactive flags per consul.
// These are used when ExtraArgs is not set in the consul's config block.
var defaultExtraArgs = map[string][]string{
	"gemini":  {"--yolo"},
	"copilot": {"--allow-all-tools"},
	"oz":      {"--no-interactive"},
	"claude":  {"--dangerously-skip-permissions"},
	"codex":   {"-q"},
}

// DefaultExtraArgs returns the built-in non-interactive flags for consulName.
// Returns nil for unknown consul names.
func DefaultExtraArgs(consulName string) []string {
	return defaultExtraArgs[consulName]
}

// ConsulConfig holds per-consul configuration.
// Authentication is the responsibility of the user (e.g. `gh auth login`,
// `gemini auth`). No API keys are stored here.
type ConsulConfig struct {
	Enabled bool   `mapstructure:"enabled"`
	Model   string `mapstructure:"model"`
	// ExtraArgs is an optional list of additional CLI flags passed to the
	// consul binary. When set it REPLACES the built-in non-interactive flag
	// defaults for that consul. Leave unset to use the defaults.
	ExtraArgs []string `mapstructure:"extra_args"`
}

// Config is the root configuration structure.
// Each consul is a named top-level field, which allows adding consul-specific
// configuration in the future without affecting other consul configs.
type Config struct {
	// Repo is the full GitHub repository path including hostname,
	// e.g. "github.com/owner/repo". Required — can be set here or
	// overridden at runtime with the --repo flag.
	Repo string `mapstructure:"repo"`

	// CodeReviewSkill is optional. When empty the binary's bundled skill is
	// used. Can be overridden at runtime by the --skill flag.
	CodeReviewSkill string       `mapstructure:"code_review_skill"`
	Gemini          ConsulConfig `mapstructure:"gemini"`
	Copilot         ConsulConfig `mapstructure:"copilot"`
	Oz              ConsulConfig `mapstructure:"oz"`
	Claude          ConsulConfig `mapstructure:"claude"`
	Codex           ConsulConfig `mapstructure:"codex"`
}

// EnabledConsuls returns a map of consul name → ConsulConfig for every consul
// that has Enabled = true. This gives the orchestrator a stable iteration
// interface regardless of the concrete per-consul field layout.
func (c *Config) EnabledConsuls() map[string]ConsulConfig {
	all := map[string]ConsulConfig{
		"gemini":  c.Gemini,
		"copilot": c.Copilot,
		"oz":      c.Oz,
		"claude":  c.Claude,
		"codex":   c.Codex,
	}
	enabled := make(map[string]ConsulConfig, len(all))
	for name, cc := range all {
		if cc.Enabled {
			enabled[name] = cc
		}
	}
	return enabled
}

// Load reads the YAML config at path (expanding ~ first) and unmarshals it.
func Load(path string) (*Config, error) {
	expanded, err := expandTilde(path)
	if err != nil {
		return nil, fmt.Errorf("expanding config path: %w", err)
	}

	v := viper.New()
	v.SetConfigFile(expanded)
	v.SetConfigType("yaml")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("reading config %q: %w", expanded, err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshaling config: %w", err)
	}

	cfg.CodeReviewSkill, err = expandTilde(cfg.CodeReviewSkill)
	if err != nil {
		return nil, fmt.Errorf("expanding code_review_skill path: %w", err)
	}

	return &cfg, nil
}

// Validate performs structural validation of the config.
//   - repo must not be empty
//   - code_review_skill, if set, must exist on disk (empty = use bundled default)
//   - At least one consul must be enabled
func Validate(c *Config) error {
	var errs []string

	// code_review_skill is optional — empty means use the bundled default.
	if c.CodeReviewSkill != "" {
		if _, err := os.Stat(c.CodeReviewSkill); err != nil {
			errs = append(errs, fmt.Sprintf("code_review_skill %q not found: %v", c.CodeReviewSkill, err))
		}
	}

	if len(c.EnabledConsuls()) == 0 {
		errs = append(errs, "at least one consul must be enabled")
	}

	if len(errs) > 0 {
		return errors.New(strings.Join(errs, "; "))
	}
	return nil
}

// LookPathFunc is the function used to locate binaries.
// It is a package-level variable so tests can inject a fake.
var LookPathFunc = exec.LookPath

// CheckBinaries verifies that every enabled consul's binary is present in PATH.
// No other binaries are required — each agent handles PR retrieval itself,
// as instructed by the skill file.
func CheckBinaries(c *Config) error {
	var missing []string
	for name := range c.EnabledConsuls() {
		bin := SupportedConsuls[name]
		if _, err := LookPathFunc(bin); err != nil {
			missing = append(missing, bin)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("required binaries not found in PATH: %s", strings.Join(missing, ", "))
	}
	return nil
}

// expandTilde replaces a leading ~ with the user's home directory.
func expandTilde(path string) (string, error) {
	if !strings.HasPrefix(path, "~") {
		return path, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolving home directory: %w", err)
	}
	return filepath.Join(home, path[1:]), nil
}
