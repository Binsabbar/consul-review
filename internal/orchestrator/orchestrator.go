// Package orchestrator coordinates parallel AI consul code reviews.
package orchestrator

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
	"sync"

	"github.com/binsabbar/consul-review/internal/agent"
)

// buildPrompt constructs the prompt passed to each agent.
// The skill file content provides instructions on HOW to review (including
// how to fetch the PR — via gh CLI, MCP, API, etc.). The repo and PR number
// tell the agent WHAT to review. The agent binary follows the skill to decide
// how it fetches the PR details.
func buildPrompt(skillContent, repo, prNumber string) string {
	return fmt.Sprintf("Use the following skill to review the PR:\n%s\n\nRepository: %s\nPR: #%s\n", skillContent, repo, prNumber)
}

// workerResult holds the review output and any error for a single agent run.
type workerResult struct {
	agentName string
	output    string
	err       error
}

// Orchestrate fans the review request out to all agents in parallel and prints
// each agent's review to stdout when all agents have finished.
//
// The orchestrator is intentionally thin — it has no knowledge of how each
// agent fetches PR details. The skill file instructs each agent binary on
// how to retrieve and review the PR (via gh CLI, MCP server, API, etc.).
//
// Concurrency: each agent runs in its own goroutine controlled by a WaitGroup;
// results are collected under a Mutex so no goroutine blocks another.
func Orchestrate(ctx context.Context, agents []agent.Agent, skillContent, repo, prNumber string) error {
	if len(agents) == 0 {
		return fmt.Errorf("no agents provided")
	}

	prompt := buildPrompt(skillContent, repo, prNumber)
	req := agent.ReviewRequest{
		Prompt:   prompt,
		Repo:     repo,
		PRNumber: prNumber,
	}

	fmt.Fprintf(os.Stderr, "🚀 Starting review for PR #%s on %s\n", prNumber, repo)
	slog.Info("starting review", "repo", repo, "pr", prNumber, "agents", len(agents))

	results := make([]workerResult, 0, len(agents))
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	for _, ag := range agents {
		ag := ag //nolint:copyloopvar // intentional: Go < 1.22 compat inside goroutine

		wg.Add(1)
		go func() {
			defer wg.Done()

			slog.Info("starting consul", "consul", ag.Name())
			fmt.Fprintf(os.Stderr, "⏳ Agent %s is reviewing...\n", ag.Name())

			res, err := ag.Review(ctx, req)

			slog.Info("consul finished", "consul", ag.Name(), "err", err)
			if err != nil {
				fmt.Fprintf(os.Stderr, "❌ Agent %s failed: %v\n", ag.Name(), err)
			} else {
				fmt.Fprintf(os.Stderr, "✅ Agent %s finished.\n", ag.Name())
			}

			mu.Lock()
			results = append(results, workerResult{
				agentName: ag.Name(),
				output:    res.Output,
				err:       err,
			})
			mu.Unlock()
		}()
	}

	wg.Wait()

	// Print all results to stdout and collect errors.
	var errs []string
	for _, res := range results {
		if res.err != nil {
			slog.Error("consul failed", "consul", res.agentName, "err", res.err)
			errs = append(errs, fmt.Sprintf("%s: %v", res.agentName, res.err))
			continue
		}

		_, _ = fmt.Fprintf(os.Stdout, "\n%s\n%s\n%s\n\n%s\n",
			strings.Repeat("=", 60),
			fmt.Sprintf("Review by: %s", res.agentName),
			strings.Repeat("=", 60),
			res.output,
		)
	}

	if len(errs) > 0 && len(errs) == len(agents) {
		return fmt.Errorf("all agents failed: %s", strings.Join(errs, "; "))
	}
	if len(errs) > 0 {
		return fmt.Errorf("partial failure — %d/%d agents errored: %s",
			len(errs), len(agents), strings.Join(errs, "; "))
	}
	return nil
}
