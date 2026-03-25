// Package cmd provides the consul-review CLI commands.
package cmd

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"text/template"

	"github.com/spf13/cobra"
)

// configInitTemplate is the default config file written by `consul-review config init`.
const configInitTemplate = `# consul-review configuration
# Reference: https://github.com/binsabbar/consul-review
#
# Authentication: configure each agent's CLI auth BEFORE running consul-review.
#   gemini:  gemini auth login
#   copilot: gh auth login  (or set GH_TOKEN)
#   oz:      oz auth login
#   codex:   codex auth login  (or set OPENAI_API_KEY)
#   claude:  claude auth login  (or set ANTHROPIC_API_KEY)
#
# Full GitHub repository path including hostname (required).
# Can also be passed at runtime via: consul-review review --repo github.com/owner/repo
repo: "github.com/owner/repo"

# Skill file (optional): path to a custom SKILL.md for the code review agent.
# If omitted, the bundled default go-code-review skill is used.
# Can also be passed at runtime via: consul-review review --skill /path/to/SKILL.md
# code_review_skill: "~/.agents/skills/my-custom-review/SKILL.md"

# ─── Consuls ──────────────────────────────────────────────────────────────────
# Enable the agents you have installed and authenticated.
# extra_args is optional — when set it REPLACES the built-in non-interactive
# flags for that consul. Omit extra_args to use the defaults shown below:
#   gemini:  [--yolo]
#   copilot: [--allow-all-tools]
#   oz:      [--no-interactive]
#   codex:   [--full-auto]
#   claude:  [--dangerously-skip-permissions]

gemini:
  enabled: true
  model: "gemini-2.5-pro"
  # extra_args: ["--yolo", "--model", "gemini-2.5-pro"]

copilot:
  enabled: false
  model: "gpt-4"
  # extra_args: ["--allow-all-tools"]

oz:
  enabled: false
  model: "claude-3-5-sonnet"
  # extra_args: ["--no-interactive"]

codex:
  enabled: false
  model: "gpt-5.3-codex"
  # extra_args: ["--full-auto"]

claude:
  enabled: false
  model: "claude-sonnet-4-6"
  # extra_args: ["--dangerously-skip-permissions"]
`

// configCmd is the parent for config-management subcommands.
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Manage consul-review configuration",
}

// configInitCmd initialises the default config file.
var configInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialise the default config file (~/.consul-review/config.yaml)",
	Long: `Creates ~/.consul-review/config.yaml with sensible defaults.
Exits with an error if the file already exists unless --force is given.`,
	RunE: runConfigInit,
}

var forceInit bool

func init() {
	configInitCmd.Flags().BoolVar(&forceInit, "force", false, "overwrite an existing config file")
	configCmd.AddCommand(configInitCmd)
	rootCmd.AddCommand(configCmd)
}

func runConfigInit(_ *cobra.Command, _ []string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("resolving home directory: %w", err)
	}

	dir := filepath.Join(home, ".consul-review")
	dest := filepath.Join(dir, "config.yaml")

	// Guard against accidental overwrites.
	if _, err := os.Stat(dest); err == nil && !forceInit {
		return fmt.Errorf("config file already exists at %s (use --force to overwrite)", dest)
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return fmt.Errorf("checking config path: %w", err)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}

	f, err := os.OpenFile(dest, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o600) //nolint:gosec // fixed path under user home dir
	if err != nil {
		return fmt.Errorf("creating config file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to close config file: %v\n", cerr)
		}
	}()

	tmpl, err := template.New("config").Parse(configInitTemplate)
	if err != nil {
		return fmt.Errorf("parsing config template: %w", err)
	}
	if err := tmpl.Execute(f, nil); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	fmt.Printf("✅ Config initialised at %s\n", dest)
	fmt.Println("Edit the file to enable your preferred consuls, then run:")
	fmt.Printf("   consul-review review --pr <PR_NUMBER>\n")
	return nil
}
