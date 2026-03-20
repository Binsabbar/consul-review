// Package runner provides an abstraction for executing external commands.
package runner

import (
	"context"
	"io"
	"os/exec"
)

// Runner executes an external binary and streams its stdout to the provided writer.
// Implementing this interface for each production binary or a fake in tests keeps
// subprocess logic fully decoupled from orchestration logic.
type Runner interface {
	Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout io.Writer) error
}

// OSRunner is the production Runner that shells out via os/exec.
type OSRunner struct{}

// Run executes the binary at name with args, wiring stdin and stdout. Stderr is
// discarded; callers should redirect it if desired. The command is cancelled when
// ctx is done.
func (OSRunner) Run(ctx context.Context, name string, args []string, stdin io.Reader, stdout io.Writer) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	return cmd.Run()
}
