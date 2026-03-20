package orchestrator

import (
	"context"
	"fmt"
	"io"
	"sync"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/binsabbar/consul-review/internal/agent"
	"github.com/binsabbar/consul-review/internal/runner"
)

// OrchestratorSuite tests the orchestration logic.
type OrchestratorSuite struct {
	suite.Suite
}

func TestOrchestratorSuite(t *testing.T) {
	suite.Run(t, new(OrchestratorSuite))
}

// ---------------------------------------------------------------------------
// buildPrompt
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestBuildPrompt_ContainsAllSections() {
	prompt := buildPrompt("## Skill", "Fix auth bug", "Fixes a security issue.", "- old\n+ new")
	s.Require().Contains(prompt, "## Skill")
	s.Require().Contains(prompt, "Fix auth bug")
	s.Require().Contains(prompt, "Fixes a security issue.")
	s.Require().Contains(prompt, "- old\n+ new")
}

func (s *OrchestratorSuite) TestBuildPrompt_EmptyBodyOmitted() {
	prompt := buildPrompt("skill", "title", "", "diff")
	s.Require().NotContains(prompt, "Description")
}

// ---------------------------------------------------------------------------
// parsePRMeta
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestParsePRMeta() {
	m := parsePRMeta("TITLE:Fix the bug\nBODY:Detailed description")
	s.Require().Equal("Fix the bug", m.Title)
	s.Require().Equal("Detailed description", m.Body)
}

// ---------------------------------------------------------------------------
// Orchestrate — parallel agent execution
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestOrchestrate_AllAgentsRunInParallel() {
	gemini := &fakeAgent{name: "gemini", output: "gemini review"}
	copilot := &fakeAgent{name: "copilot", output: "copilot review"}

	ghRunner := newGhFake("TITLE:Test PR\nBODY:body", "+ line")

	err := Orchestrate(context.Background(),
		[]agent.Agent{gemini, copilot},
		"## skill", "42", ghRunner)

	s.Require().NoError(err)
	s.Require().True(gemini.called, "gemini must be called")
	s.Require().True(copilot.called, "copilot must be called")
	s.Require().True(ghRunner.calledWith("claude"), "aggregation must run")
}

func (s *OrchestratorSuite) TestOrchestrate_PartialFailure_OthersContinue() {
	good := &fakeAgent{name: "copilot", output: "copilot review"}
	bad := &fakeAgent{name: "gemini", err: fmt.Errorf("simulated crash")}

	ghRunner := newGhFake("TITLE:Test PR\nBODY:body", "+ line")

	err := Orchestrate(context.Background(),
		[]agent.Agent{bad, good},
		"## skill", "99", ghRunner)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "partial failure")
	s.Require().True(good.called, "good agent must still run")
	s.Require().True(ghRunner.calledWith("claude"), "aggregation runs with the surviving output")
}

func (s *OrchestratorSuite) TestOrchestrate_AllFail_ReturnsError() {
	ghRunner := newGhFake("TITLE:PR\nBODY:", "+ line")

	err := Orchestrate(context.Background(),
		[]agent.Agent{
			&fakeAgent{name: "gemini", err: fmt.Errorf("fail")},
			&fakeAgent{name: "copilot", err: fmt.Errorf("fail")},
		},
		"## skill", "1", ghRunner)

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "all consuls failed")
}

// ---------------------------------------------------------------------------
// Fake agent (satisfies agent.Agent)
// ---------------------------------------------------------------------------

type fakeAgent struct {
	name   string
	output string
	err    error
	called bool
}

func (f *fakeAgent) Name() string { return f.name }

func (f *fakeAgent) Review(_ context.Context, _ agent.ReviewRequest) (agent.ReviewResult, error) {
	f.called = true
	return agent.ReviewResult{AgentName: f.name, Output: f.output}, f.err
}

// ---------------------------------------------------------------------------
// Fake gh runner (only handles gh + claude calls)
// ---------------------------------------------------------------------------

type ghFakeRunner struct {
	mu    sync.Mutex
	meta  string
	diff  string
	calls []string
}

func newGhFake(meta, diff string) *ghFakeRunner {
	return &ghFakeRunner{meta: meta, diff: diff}
}

func (g *ghFakeRunner) Run(_ context.Context, name string, args []string, _ io.Reader, stdout io.Writer) error {
	g.mu.Lock()
	g.calls = append(g.calls, name)
	g.mu.Unlock()

	if name == "gh" {
		for _, a := range args {
			if a == "view" {
				_, _ = fmt.Fprint(stdout, g.meta)
				return nil
			}
		}
		_, _ = fmt.Fprint(stdout, g.diff)
	}
	return nil
}

func (g *ghFakeRunner) calledWith(name string) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	for _, c := range g.calls {
		if c == name {
			return true
		}
	}
	return false
}

// Compile-time check.
var _ runner.Runner = (*ghFakeRunner)(nil)
