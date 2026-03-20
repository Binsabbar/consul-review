package runner

import (
	"context"
	"fmt"
	"io"
	"sync"
)

// Call records a single invocation made through FakeRunner.
type Call struct {
	Name string
	Args []string
}

// FakeRunner is a deterministic test double for Runner.
// It records every call and writes a configurable Output to stdout.
// If Err is non-nil it is returned from Run (after writing Output).
type FakeRunner struct {
	mu     sync.Mutex
	Calls  []Call
	Output string
	Err    error
}

// Run records the call, writes Output to stdout, then returns Err.
func (f *FakeRunner) Run(_ context.Context, name string, args []string, _ io.Reader, stdout io.Writer) error {
	f.mu.Lock()
	f.Calls = append(f.Calls, Call{Name: name, Args: args})
	f.mu.Unlock()

	if f.Output != "" {
		if _, err := fmt.Fprint(stdout, f.Output); err != nil {
			return err
		}
	}
	return f.Err
}

// CallCount returns the number of times Run was invoked. Safe for concurrent use.
func (f *FakeRunner) CallCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return len(f.Calls)
}

// CalledWith returns true if the runner was called with the given binary name.
func (f *FakeRunner) CalledWith(name string) bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	for _, c := range f.Calls {
		if c.Name == name {
			return true
		}
	}
	return false
}
