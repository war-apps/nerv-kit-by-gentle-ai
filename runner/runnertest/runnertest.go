// Package runnertest provides FakeRunner, the scripted test double for
// runner.Runner: tests that need a Runner script its responses here
// instead of launching a real process.
package runnertest

import (
	"context"
	"strings"
)

// Call records one FakeRunner.Run invocation.
type Call struct {
	Name string
	Args []string
}

// Response is one scripted runner.Runner.Run result.
type Response struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Err      error
}

// FakeRunner is a scripted runner.Runner. Responses is keyed first by the
// full command line ("name arg1 arg2 ..."), then by name alone, then
// falls back to Default. Every call is recorded in Calls regardless of
// whether a response was found.
type FakeRunner struct {
	Responses map[string]Response
	Default   Response
	Calls     []Call
}

// Run implements runner.Runner.
func (f *FakeRunner) Run(_ context.Context, name string, args ...string) (stdout, stderr string, exitCode int, err error) {
	f.Calls = append(f.Calls, Call{Name: name, Args: append([]string(nil), args...)})

	if r, ok := f.Responses[commandLine(name, args)]; ok {
		return r.Stdout, r.Stderr, r.ExitCode, r.Err
	}
	if r, ok := f.Responses[name]; ok {
		return r.Stdout, r.Stderr, r.ExitCode, r.Err
	}
	return f.Default.Stdout, f.Default.Stderr, f.Default.ExitCode, f.Default.Err
}

func commandLine(name string, args []string) string {
	if len(args) == 0 {
		return name
	}
	return name + " " + strings.Join(args, " ")
}
