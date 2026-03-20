package orchestrator

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/suite"

	"github.com/binsabbar/consul-review/internal/agent"
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

func (s *OrchestratorSuite) TestBuildPrompt_ContainsAllParts() {
	prompt := buildPrompt("## Skill\nDo a great review.", "github.com/owner/repo", "42")
	s.Require().Contains(prompt, "## Skill")
	s.Require().Contains(prompt, "github.com/owner/repo")
	s.Require().Contains(prompt, "#42")
}

// ---------------------------------------------------------------------------
// Orchestrate — parallel agent execution
// ---------------------------------------------------------------------------

func (s *OrchestratorSuite) TestOrchestrate_AllAgentsRunInParallel() {
	gemini := &fakeAgent{name: "gemini", output: "gemini review"}
	copilot := &fakeAgent{name: "copilot", output: "copilot review"}

	err := Orchestrate(context.Background(),
		[]agent.Agent{gemini, copilot},
		"## skill", "github.com/test/repo", "42")

	s.Require().NoError(err)
	s.Require().True(gemini.called, "gemini must be called")
	s.Require().True(copilot.called, "copilot must be called")
}

func (s *OrchestratorSuite) TestOrchestrate_PartialFailure_OthersContinue() {
	good := &fakeAgent{name: "copilot", output: "copilot review"}
	bad := &fakeAgent{name: "gemini", err: fmt.Errorf("simulated crash")}

	err := Orchestrate(context.Background(),
		[]agent.Agent{bad, good},
		"## skill", "github.com/test/repo", "99")

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "partial failure")
	s.Require().True(good.called, "good agent must still run")
}

func (s *OrchestratorSuite) TestOrchestrate_AllFail_ReturnsError() {
	err := Orchestrate(context.Background(),
		[]agent.Agent{
			&fakeAgent{name: "gemini", err: fmt.Errorf("fail")},
			&fakeAgent{name: "copilot", err: fmt.Errorf("fail")},
		},
		"## skill", "github.com/test/repo", "1")

	s.Require().Error(err)
	s.Require().Contains(err.Error(), "all agents failed")
}

func (s *OrchestratorSuite) TestOrchestrate_NoAgents_ReturnsError() {
	err := Orchestrate(context.Background(), nil, "## skill", "github.com/test/repo", "1")
	s.Require().Error(err)
}

func (s *OrchestratorSuite) TestOrchestrate_PromptContainsRepoAndPR() {
	var capturedPrompt string
	capture := &capturingAgent{name: "test", onReview: func(req agent.ReviewRequest) {
		capturedPrompt = req.Prompt
	}}

	_ = Orchestrate(context.Background(),
		[]agent.Agent{capture},
		"## My Skill", "github.com/owner/repo", "99")

	s.Require().Contains(capturedPrompt, "github.com/owner/repo")
	s.Require().Contains(capturedPrompt, "#99")
}

func (s *OrchestratorSuite) TestOrchestrate_RequestHasRepoAndPRNumber() {
	var capturedReq agent.ReviewRequest
	capture := &capturingAgent{name: "test", onReview: func(req agent.ReviewRequest) {
		capturedReq = req
	}}

	_ = Orchestrate(context.Background(),
		[]agent.Agent{capture},
		"## Skill", "github.com/owner/repo", "55")

	s.Require().Equal("github.com/owner/repo", capturedReq.Repo)
	s.Require().Equal("55", capturedReq.PRNumber)
}

// ---------------------------------------------------------------------------
// Fake agents
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

type capturingAgent struct {
	name     string
	onReview func(agent.ReviewRequest)
}

func (c *capturingAgent) Name() string { return c.name }

func (c *capturingAgent) Review(_ context.Context, req agent.ReviewRequest) (agent.ReviewResult, error) {
	c.onReview(req)
	return agent.ReviewResult{AgentName: c.name, Output: "ok"}, nil
}

// Keep strings import used in test output assertions.
var _ = strings.Contains
