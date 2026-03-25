// Package binary provides a binary-backed Agent implementation that shells out
// to a locally installed consul CLI (gemini, copilot, oz, codex, claude).
//
// This is the current (v0.x) implementation. When direct API backends are
// ready, they will live in sibling packages (e.g. internal/agent/anthropic,
// internal/agent/gemini) and satisfy the same agent.Agent interface without
// any changes to the orchestrator or CLI layer.
package binary

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/binsabbar/consul-review/internal/agent"
	"github.com/binsabbar/consul-review/internal/config"
	"github.com/binsabbar/consul-review/internal/runner"
)

// consulDef describes a supported consul binary.
//
// promptArgs encodes how the binary expects the prompt (flag + value).
// It is always used — even when extraArgs overrides the non-interactive flags —
// so the prompt is never appended as a bare positional argument.
type consulDef struct {
	bin         string
	promptArgs  func(prompt string) []string
	defaultArgs func(model, prompt string) []string
}

// supportedConsuls holds the built-in argument builders for each consul.
var supportedConsuls = map[string]consulDef{
	"gemini": {
		bin:        "gemini",
		promptArgs: func(prompt string) []string { return []string{"-p", prompt} },
		defaultArgs: func(model, prompt string) []string {
			return []string{"-p", prompt, "--yolo", "--model", model}
		},
	},
	"copilot": {
		bin:        "copilot",
		promptArgs: func(prompt string) []string { return []string{"-p", prompt} },
		defaultArgs: func(model, prompt string) []string {
			return []string{"-p", prompt, "--allow-all-tools", "--model", model}
		},
	},
	"oz": {
		bin:        "oz",
		promptArgs: func(prompt string) []string { return []string{"agent", "run", "--prompt", prompt} },
		defaultArgs: func(model, prompt string) []string {
			return []string{"agent", "run", "--prompt", prompt, "--no-interactive", "--model", model}
		},
	},
	"codex": {
		bin:        "codex",
		promptArgs: func(prompt string) []string { return []string{"-p", prompt} },
		defaultArgs: func(model, prompt string) []string {
			return []string{"-p", prompt, "--full-auto", "--model", model}
		},
	},
	"claude": {
		bin:        "claude",
		promptArgs: func(prompt string) []string { return []string{"-p", prompt} },
		defaultArgs: func(model, prompt string) []string {
			return []string{"-p", prompt, "--dangerously-skip-permissions", "--model", model}
		},
	},
}

// argsFor builds the CLI arguments for the named consul.
//
// When extraArgs is non-empty it replaces the non-interactive/model flags, but
// the prompt is ALWAYS passed via the consul's own prompt flag (promptArgs) —
// never as a bare positional argument.
//
//	result = [promptArgs...] + [extraArgs...]
func argsFor(consulName, model, prompt string, extraArgs []string) (bin string, args []string, err error) {
	def, ok := supportedConsuls[consulName]
	if !ok {
		return "", nil, fmt.Errorf("unknown consul %q: no binary definition found", consulName)
	}
	if len(extraArgs) > 0 {
		args = append(def.promptArgs(prompt), extraArgs...)
	} else {
		args = def.defaultArgs(model, prompt)
	}
	return def.bin, args, nil
}

// Agent shells out to a locally installed consul binary to perform a review.
// It satisfies the agent.Agent interface and can be swapped for an API-backed
// implementation with no changes to the orchestrator.
type Agent struct {
	name   string
	cfg    config.ConsulConfig
	runner runner.Runner
}

// New constructs a binary Agent for the given consul name and config.
// The runner is used to execute the consul binary; use runner.OSRunner{} in
// production and a fake runner in tests.
func New(name string, cfg config.ConsulConfig, r runner.Runner) *Agent {
	return &Agent{name: name, cfg: cfg, runner: r}
}

// Name returns the consul's identifier (e.g. "gemini").
func (a *Agent) Name() string { return a.name }

// Review runs the consul binary with the constructed prompt and returns the
// full review output. ExtraArgs in the request take precedence over config;
// config ExtraArgs take precedence over the built-in defaults.
func (a *Agent) Review(ctx context.Context, req agent.ReviewRequest) (agent.ReviewResult, error) {
	// Priority: request ExtraArgs > config ExtraArgs > built-in defaults.
	extraArgs := req.ExtraArgs
	if len(extraArgs) == 0 {
		extraArgs = a.cfg.ExtraArgs
	}

	model := req.Model
	if model == "" {
		model = a.cfg.Model
	}

	bin, args, err := argsFor(a.name, model, req.Prompt, extraArgs)
	if err != nil {
		return agent.ReviewResult{AgentName: a.name}, err
	}

	var buf bytes.Buffer
	var out io.Writer = &buf
	if req.Stream != nil {
		out = io.MultiWriter(&buf, req.Stream)
	}

	if err := a.runner.Run(ctx, bin, args, nil, out); err != nil {
		return agent.ReviewResult{AgentName: a.name}, fmt.Errorf("running %s: %w", bin, err)
	}

	return agent.ReviewResult{
		AgentName: a.name,
		Output:    buf.String(),
	}, nil
}
