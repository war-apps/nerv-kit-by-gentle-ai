package runnertest_test

import (
	"context"
	"errors"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/runner"
	"github.com/war-apps/nerv-kit-by-gentle-ai/runner/runnertest"
)

// Compile-time proof FakeRunner satisfies runner.Runner without importing
// it from a variable declaration inside a test (kept here, next to the
// tests that exercise it).
var _ runner.Runner = (*runnertest.FakeRunner)(nil)

func TestFakeRunner_ReturnsScriptedResponseByExactCommand(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude --version": {Stdout: "claude 4.0.0\n", ExitCode: 0},
		},
	}

	stdout, _, exitCode, err := r.Run(context.Background(), "claude", "--version")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if exitCode != 0 || stdout != "claude 4.0.0\n" {
		t.Fatalf("Run() = (%q, _, %d), want scripted response", stdout, exitCode)
	}
}

func TestFakeRunner_FallsBackToNameOnlyKey(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"npx": {ExitCode: 0},
		},
	}

	_, _, exitCode, err := r.Run(context.Background(), "npx", "skills", "add", "mattpocock/skills")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if exitCode != 0 {
		t.Fatalf("Run() exitCode = %d, want 0 from the name-only fallback", exitCode)
	}
}

func TestFakeRunner_UsesDefaultWhenNoResponseMatches(t *testing.T) {
	r := &runnertest.FakeRunner{
		Default: runnertest.Response{ExitCode: 1, Err: errors.New("boom")},
	}

	_, _, exitCode, err := r.Run(context.Background(), "unscripted", "cmd")
	if err == nil || exitCode != 1 {
		t.Fatalf("Run() = (_, _, %d, %v), want the Default response", exitCode, err)
	}
}

func TestFakeRunner_RecordsEveryCall(t *testing.T) {
	r := &runnertest.FakeRunner{}

	_, _, _, _ = r.Run(context.Background(), "claude", "--version")
	_, _, _, _ = r.Run(context.Background(), "npx", "skills", "add", "repo")

	if len(r.Calls) != 2 {
		t.Fatalf("len(Calls) = %d, want 2", len(r.Calls))
	}
	if r.Calls[0].Name != "claude" || len(r.Calls[0].Args) != 1 || r.Calls[0].Args[0] != "--version" {
		t.Fatalf("Calls[0] = %+v, unexpected", r.Calls[0])
	}
	if r.Calls[1].Name != "npx" || len(r.Calls[1].Args) != 3 {
		t.Fatalf("Calls[1] = %+v, unexpected", r.Calls[1])
	}
}
