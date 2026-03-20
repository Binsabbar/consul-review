// Package orchestrator coordinates parallel AI consul code reviews.
package orchestrator

import (
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/binsabbar/consul-review/internal/agent"
	"github.com/binsabbar/consul-review/internal/runner"
)

// buildPrompt combines the skill file content and PR metadata into a single
// prompt string that is passed verbatim to each consul agent.
func buildPrompt(skillContent, prTitle, prBody, diff string) string {
	var sb strings.Builder
	sb.WriteString("## Code Review Skill\n\n")
	sb.WriteString(skillContent)
	sb.WriteString("\n\n## Pull Request\n\n")
	sb.WriteString(fmt.Sprintf("**Title**: %s\n\n", prTitle))
	if prBody != "" {
		sb.WriteString(fmt.Sprintf("**Description**:\n%s\n\n", prBody))
	}
	sb.WriteString("## Diff\n\n```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```\n")
	return sb.String()
}

// PRMeta holds the data fetched from GitHub via the gh CLI.
type PRMeta struct {
	Title string
	Body  string
	Diff  string
}

// fetchPR retrieves the PR title, body, and unified diff for prNumber using
// the gh CLI through the Runner interface. repo must be the full path
// including hostname, e.g. "github.com/owner/repo".
func fetchPR(ctx context.Context, r runner.Runner, repo, prNumber string) (PRMeta, error) {
	var metaBuf bytes.Buffer
	if err := r.Run(ctx, "gh", []string{
		"pr", "view", prNumber,
		"--repo", repo,
		"--json", "title,body",
		"--jq", `"TITLE:\(.title)\nBODY:\(.body)"`,
	}, nil, &metaBuf); err != nil {
		return PRMeta{}, fmt.Errorf("gh pr view: %w", err)
	}

	var diffBuf bytes.Buffer
	if err := r.Run(ctx, "gh", []string{"pr", "diff", prNumber, "--repo", repo}, nil, &diffBuf); err != nil {
		return PRMeta{}, fmt.Errorf("gh pr diff: %w", err)
	}

	meta := parsePRMeta(metaBuf.String())
	meta.Diff = diffBuf.String()
	return meta, nil
}

// parsePRMeta extracts title and body from the TITLE:/BODY: format.
func parsePRMeta(raw string) PRMeta {
	var m PRMeta
	for _, line := range strings.SplitN(raw, "\n", 3) {
		switch {
		case strings.HasPrefix(line, "TITLE:"):
			m.Title = strings.TrimPrefix(line, "TITLE:")
		case strings.HasPrefix(line, "BODY:"):
			m.Body = strings.TrimPrefix(line, "BODY:")
		}
	}
	return m
}

// workerResult holds the temp-file path and any error for a single consul run.
type workerResult struct {
	agentName  string
	outputFile string
	err        error
}

// Orchestrate is the main entry point.
//
// It fetches PR data via gh, fans the review out to all provided agents in
// parallel, then runs Claude to aggregate the outputs into a consolidated
// review printed to stdout.
//
// repo must be the full GitHub path including hostname, e.g.
// "github.com/owner/repo". This is passed to the gh CLI and embedded in
// ReviewRequest so future API agents can use it directly.
func Orchestrate(ctx context.Context, agents []agent.Agent, skillContent, repo, prNumber string, r runner.Runner) error {
	slog.Info("fetching PR", "repo", repo, "pr", prNumber)
	pr, err := fetchPR(ctx, r, repo, prNumber)
	if err != nil {
		return fmt.Errorf("fetching PR #%s: %w", prNumber, err)
	}

	prompt := buildPrompt(skillContent, pr.Title, pr.Body, pr.Diff)
	req := agent.ReviewRequest{Prompt: prompt, Repo: repo}

	outDir, err := os.MkdirTemp("", "consul-review-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	slog.Info("review outputs will be written to", "dir", outDir)

	results := make([]workerResult, 0, len(agents))
	var (
		wg sync.WaitGroup
		mu sync.Mutex
	)

	for _, ag := range agents {
		wg.Add(1)
		go func() {
			defer wg.Done()

			slog.Info("starting consul", "consul", ag.Name())
			res, reviewErr := ag.Review(ctx, req)

			wr := workerResult{agentName: ag.Name(), err: reviewErr}
			if reviewErr == nil {
				wr.outputFile, reviewErr = writeOutput(outDir, ag.Name(), res.Output)
				if reviewErr != nil {
					wr.err = reviewErr
				}
			}
			if wr.err == nil {
				slog.Info("consul finished", "consul", ag.Name(), "output", wr.outputFile)
			}

			mu.Lock()
			results = append(results, wr)
			mu.Unlock()
		}()
	}

	wg.Wait()

	var outputFiles, runErrs []string
	for _, res := range results {
		if res.err != nil {
			slog.Error("consul failed", "consul", res.agentName, "err", res.err)
			runErrs = append(runErrs, fmt.Sprintf("%s: %v", res.agentName, res.err))
		} else {
			outputFiles = append(outputFiles, res.outputFile)
		}
	}

	if len(outputFiles) == 0 {
		return fmt.Errorf("all consuls failed: %s", strings.Join(runErrs, "; "))
	}

	if err := runAggregation(ctx, r, outputFiles); err != nil {
		return fmt.Errorf("aggregation step: %w", err)
	}

	if len(runErrs) > 0 {
		return fmt.Errorf("partial failure — some consuls errored: %s", strings.Join(runErrs, "; "))
	}
	return nil
}

// writeOutput writes review content to a temp file and returns its path.
func writeOutput(outDir, name, content string) (string, error) {
	outPath := filepath.Join(outDir, fmt.Sprintf("review_%s.md", name))
	f, err := os.Create(outPath) //nolint:gosec // path is constructed from trusted inputs (temp dir + agent name)
	if err != nil {
		return "", fmt.Errorf("creating output file: %w", err)
	}
	defer func() {
		if cerr := f.Close(); cerr != nil {
			slog.Warn("failed to close review output file", "path", outPath, "err", cerr)
		}
	}()

	if _, err := fmt.Fprint(f, content); err != nil {
		return "", fmt.Errorf("writing review output: %w", err)
	}
	return outPath, nil
}

// runAggregation calls claude with all review output files to produce a
// consolidated final review printed to stdout.
func runAggregation(ctx context.Context, r runner.Runner, outputFiles []string) error {
	prompt := fmt.Sprintf(
		"Analyze the following AI code review outputs and produce a single, concise, consolidated review summary highlighting the most critical findings, areas of agreement, and any conflicting opinions: %s",
		strings.Join(outputFiles, " "),
	)
	slog.Info("running aggregation", "files", outputFiles)
	return r.Run(ctx, "claude", []string{prompt}, nil, os.Stdout)
}
