package binary

import (
	"context"
	"fmt"
	"io"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/binsabbar/consul-review/internal/agent"
	"github.com/binsabbar/consul-review/internal/config"
	"github.com/binsabbar/consul-review/internal/runner"
)

// BinaryAgentSuite tests the binary Agent implementation.
type BinaryAgentSuite struct {
	suite.Suite
}

func TestBinaryAgentSuite(t *testing.T) {
	suite.Run(t, new(BinaryAgentSuite))
}

// ---------------------------------------------------------------------------
// argsFor — unit tests
// ---------------------------------------------------------------------------

func (s *BinaryAgentSuite) TestArgsFor_DefaultFlags() {
	cases := []struct {
		name        string
		consulName  string
		model       string
		wantBin     string
		wantContain []string
		wantErr     bool
	}{
		{
			name:        "gemini default flags include --yolo and --model",
			consulName:  "gemini",
			model:       "gemini-3-pro",
			wantBin:     "gemini",
			wantContain: []string{"-p", "--yolo", "--model", "gemini-3-pro"},
		},
		{
			name:        "copilot default flags include --allow-all-tools",
			consulName:  "copilot",
			model:       "gpt-4",
			wantBin:     "copilot",
			wantContain: []string{"-p", "--allow-all-tools"},
		},
		{
			name:        "oz default flags include agent run --prompt",
			consulName:  "oz",
			model:       "claude-4-sonnet",
			wantBin:     "oz",
			wantContain: []string{"agent", "run", "--prompt"},
		},
		{
			name:        "claude default flags include --dangerously-skip-permissions and --model",
			consulName:  "claude",
			model:       "claude-sonnet-4-6",
			wantBin:     "claude",
			wantContain: []string{"-p", "--dangerously-skip-permissions", "--model", "claude-sonnet-4-6"},
		},
		{
			name:        "codex default flags include -q and --model",
			consulName:  "codex",
			model:       "gpt-5.3-codex",
			wantBin:     "codex",
			wantContain: []string{"-q", "--model", "gpt-5.3-codex"},
		},
		{
			name:       "unknown consul returns error",
			consulName: "unknown-ai",
			wantErr:    true,
		},
	}

	for _, tc := range cases {
		s.Run(tc.name, func() {
			bin, args, err := argsFor(tc.consulName, tc.model, "review this", nil)
			if tc.wantErr {
				s.Require().Error(err)
				return
			}
			s.Require().NoError(err)
			s.Require().Equal(tc.wantBin, bin)
			for _, want := range tc.wantContain {
				s.Require().Contains(args, want)
			}
		})
	}
}

func (s *BinaryAgentSuite) TestArgsFor_ExtraArgsOverride() {
	extra := []string{"--yolo", "--sandbox"}
	_, args, err := argsFor("gemini", "gemini-3-pro", "my-prompt", extra)
	s.Require().NoError(err)
	// promptArgs for gemini = [-p, <prompt>], then extraArgs appended
	s.Require().Equal("-p", args[0], "prompt flag must come first")
	s.Require().Equal("my-prompt", args[1], "prompt value must follow its flag")
	s.Require().Contains(args, "--yolo")
	s.Require().Contains(args, "--sandbox")
	s.Require().NotContains(args, "--model", "--model is a default flag; must not appear when using extraArgs")
}

// ---------------------------------------------------------------------------
// Review — integration with fake runner
// ---------------------------------------------------------------------------

func (s *BinaryAgentSuite) TestReview_SuccessReturnsOutput() {
	fakeRun := &fakeRunner{output: "## Review\n\nLooks good."}
	ag := New("gemini", config.ConsulConfig{Model: "gemini-3-pro"}, fakeRun)

	result, err := ag.Review(context.Background(), agent.ReviewRequest{
		Prompt: "review this code",
	})

	s.Require().NoError(err)
	s.Require().Equal("gemini", result.AgentName)
	s.Require().Equal("## Review\n\nLooks good.", result.Output)
	s.Require().Equal("gemini", fakeRun.lastBin)
}

func (s *BinaryAgentSuite) TestReview_RunnerErrorPropagated() {
	fakeRun := &fakeRunner{err: fmt.Errorf("binary crashed")}
	ag := New("gemini", config.ConsulConfig{Model: "gemini-3-pro"}, fakeRun)

	_, err := ag.Review(context.Background(), agent.ReviewRequest{Prompt: "p"})
	s.Require().Error(err)
	s.Require().Contains(err.Error(), "binary crashed")
}

func (s *BinaryAgentSuite) TestReview_RequestExtraArgsOverrideConfig() {
	fakeRun := &fakeRunner{output: "ok"}
	cfgWithArgs := config.ConsulConfig{
		Model:     "gemini-3-pro",
		ExtraArgs: []string{"--config-flag"},
	}
	ag := New("gemini", cfgWithArgs, fakeRun)

	// Request-level ExtraArgs should win over config-level ExtraArgs.
	_, err := ag.Review(context.Background(), agent.ReviewRequest{
		Prompt:    "p",
		ExtraArgs: []string{"--request-flag"},
	})
	s.Require().NoError(err)
	s.Require().Contains(fakeRun.lastArgs, "--request-flag")
	s.Require().NotContains(fakeRun.lastArgs, "--config-flag")
}

func (s *BinaryAgentSuite) TestReview_ConfigExtraArgsFallback() {
	fakeRun := &fakeRunner{output: "ok"}
	cfgWithArgs := config.ConsulConfig{
		Model:     "gemini-3-pro",
		ExtraArgs: []string{"--cfg-override"},
	}
	ag := New("gemini", cfgWithArgs, fakeRun)

	// No request-level ExtraArgs → use config-level ExtraArgs.
	_, err := ag.Review(context.Background(), agent.ReviewRequest{Prompt: "p"})
	s.Require().NoError(err)
	s.Require().Contains(fakeRun.lastArgs, "--cfg-override")
}

// ---------------------------------------------------------------------------
// Fake runner
// ---------------------------------------------------------------------------

type fakeRunner struct {
	output   string
	err      error
	lastBin  string
	lastArgs []string
}

func (f *fakeRunner) Run(_ context.Context, name string, args []string, _ io.Reader, out io.Writer) error {
	f.lastBin = name
	f.lastArgs = args
	if f.err != nil {
		return f.err
	}
	_, _ = fmt.Fprint(out, f.output)
	return nil
}

// Compile-time check that fakeRunner satisfies runner.Runner.
var _ runner.Runner = (*fakeRunner)(nil)
