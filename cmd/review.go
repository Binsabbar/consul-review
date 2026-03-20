package cmd

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/binsabbar/consul-review/assets"
	"github.com/binsabbar/consul-review/internal/agent"
	binaryagent "github.com/binsabbar/consul-review/internal/agent/binary"
	"github.com/binsabbar/consul-review/internal/config"
	"github.com/binsabbar/consul-review/internal/orchestrator"
	"github.com/binsabbar/consul-review/internal/runner"
)

var (
	prNumber  string
	repoFlag  string
	skillFile string
)

// reviewCmd executes a PR review by orchestrating all enabled consuls.
var reviewCmd = &cobra.Command{
	Use:   "review",
	Short: "Review a GitHub PR using multiple AI agents",
	Long:  `Review a GitHub PR using multiple AI agents.`,
	RunE:  runReview,
}

func init() {
	reviewCmd.Flags().StringVar(&prNumber, "pr", "", "GitHub PR number to review (required)")
	_ = reviewCmd.MarkFlagRequired("pr")

	reviewCmd.Flags().StringVar(&repoFlag, "repo", "", "full GitHub repo path including hostname, e.g. github.com/owner/repo (overrides config)")
	reviewCmd.Flags().StringVar(&skillFile, "skill", "", "path to a skill file (overrides code_review_skill in config and the bundled default)")

	rootCmd.AddCommand(reviewCmd)
}

// runReview is the handler for the review subcommand.
func runReview(cmd *cobra.Command, _ []string) error {
	// 1. Resolve config path.
	configPath := cfgFile
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("resolving home dir: %w", err)
		}
		configPath = filepath.Join(home, ".consul-review", "config.yaml")
	}

	// 2. Load + validate config.
	cfg, err := config.Load(configPath)
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}
	if err := config.Validate(cfg); err != nil {
		return fmt.Errorf("invalid config: %w", err)
	}

	// 3. Pre-flight binary check.
	if err := config.CheckBinaries(cfg); err != nil {
		return fmt.Errorf("pre-flight check failed: %w", err)
	}

	// 4. Resolve repo (flag > config).
	repo, err := resolveRepo(repoFlag, cfg.Repo)
	if err != nil {
		return err
	}

	// 5. Resolve skill content (flag > config > bundled).
	skillContent, err := resolveSkill(skillFile, cfg.CodeReviewSkill)
	if err != nil {
		return err
	}

	// 6. Wire up binary agents from config.
	r := runner.OSRunner{}
	agents := makeAgents(cfg, r)

	slog.Info("starting consul-review", "repo", repo, "pr", prNumber, "agents", len(agents))

	// 7. Orchestrate.
	if err := orchestrator.Orchestrate(cmd.Context(), agents, skillContent, repo, prNumber, r); err != nil {
		return fmt.Errorf("review failed: %w", err)
	}

	slog.Info("consul-review complete", "pr", prNumber)
	return nil
}

// makeAgents constructs a binary Agent for every enabled consul in the config.
// This is the wiring point: swap binaryagent.New for an API agent constructor
// to add a cloud backend without touching the orchestrator.
func makeAgents(cfg *config.Config, r runner.Runner) []agent.Agent {
	enabled := cfg.EnabledConsuls()
	agents := make([]agent.Agent, 0, len(enabled))
	for name, cc := range enabled {
		agents = append(agents, binaryagent.New(name, cc, r))
	}
	return agents
}

// resolveSkill returns the skill content from the first non-empty source in
// priority order: CLI flag → config file path → bundled default.
func resolveSkill(flagPath, cfgPath string) (string, error) {
	switch {
	case flagPath != "":
		b, err := os.ReadFile(flagPath) //nolint:gosec // user-supplied path via --skill flag
		if err != nil {
			return "", fmt.Errorf("reading --skill file %q: %w", flagPath, err)
		}
		slog.Info("using skill from --skill flag", "path", flagPath)
		return string(b), nil

	case cfgPath != "":
		b, err := os.ReadFile(cfgPath) //nolint:gosec // validated path from config
		if err != nil {
			return "", fmt.Errorf("reading skill file %q: %w", cfgPath, err)
		}
		slog.Info("using skill from config", "path", cfgPath)
		return string(b), nil

	default:
		slog.Info("using bundled default skill (go-code-review)")
		return assets.DefaultCodeReviewSkill, nil
	}
}

// resolveRepo returns the repository path from the first non-empty source:
// --repo flag → config value. Returns an error if neither is set.
func resolveRepo(flagRepo, cfgRepo string) (string, error) {
	switch {
	case flagRepo != "":
		return flagRepo, nil
	case cfgRepo != "":
		return cfgRepo, nil
	default:
		return "", fmt.Errorf("repo is required: set it in config (repo: github.com/owner/repo) or pass --repo")
	}
}
