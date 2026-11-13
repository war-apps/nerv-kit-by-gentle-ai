package claudeplugin_test

import (
	"context"
	"strings"
	"testing"

	"github.com/war-apps/nerv-kit-by-gentle-ai/claudeplugin"
	"github.com/war-apps/nerv-kit-by-gentle-ai/runner/runnertest"
)

func TestPluginCLI_Uninstall_Success(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin uninstall kit@kit": {Stdout: "uninstalled\n", ExitCode: 0},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.Uninstall(context.Background(), "kit@kit"); err != nil {
		t.Fatalf("Uninstall() error = %v", err)
	}
	if len(r.Calls) != 1 || r.Calls[0].Name != "claude" {
		t.Fatalf("unexpected calls: %+v", r.Calls)
	}
}

func TestPluginCLI_Uninstall_TolerantOfNotInstalled(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin uninstall kit@kit": {Stderr: "Error: plugin kit@kit is not installed\n", ExitCode: 1},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.Uninstall(context.Background(), "kit@kit"); err != nil {
		t.Fatalf("Uninstall() error = %v, want nil (tolerant of 'not installed')", err)
	}
}

func TestPluginCLI_Uninstall_OtherFailureErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin uninstall kit@kit": {Stderr: "boom\n", ExitCode: 1},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	_, err := cli.Uninstall(context.Background(), "kit@kit")
	if err == nil {
		t.Fatal("Uninstall() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to mention the failure output", err)
	}
}

func TestPluginCLI_Install_Success(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin install kit@kit": {Stdout: "installed\n", ExitCode: 0},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.Install(context.Background(), "kit@kit"); err != nil {
		t.Fatalf("Install() error = %v", err)
	}
}

func TestPluginCLI_Install_FailureErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin install kit@kit": {Stderr: "network error\n", ExitCode: 1},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	_, err := cli.Install(context.Background(), "kit@kit")
	if err == nil {
		t.Fatal("Install() error = nil, want an error")
	}
}

func TestPluginCLI_LaunchFailureErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Default: runnertest.Response{Err: context.DeadlineExceeded},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.Install(context.Background(), "kit@kit"); err == nil {
		t.Fatal("Install() error = nil, want an error on launch failure")
	}
}

func TestPluginCLI_AddMarketplace_RunsExactCommand(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin marketplace add /home/u/.kit/marketplace": {Stdout: "added\n", ExitCode: 0},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.AddMarketplace(context.Background(), "/home/u/.kit/marketplace"); err != nil {
		t.Fatalf("AddMarketplace() error = %v", err)
	}
	if len(r.Calls) != 1 {
		t.Fatalf("calls = %+v, want exactly one", r.Calls)
	}
	got := r.Calls[0].Name + " " + strings.Join(r.Calls[0].Args, " ")
	if got != "claude plugin marketplace add /home/u/.kit/marketplace" {
		t.Errorf("command = %q", got)
	}
}

func TestPluginCLI_AddMarketplace_FailureErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Default: runnertest.Response{Stderr: "invalid marketplace\n", ExitCode: 1},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	_, err := cli.AddMarketplace(context.Background(), "/home/u/.kit/marketplace")
	if err == nil {
		t.Fatal("AddMarketplace() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "invalid marketplace") {
		t.Errorf("error = %v, want it to mention the failure output", err)
	}
}

func TestPluginCLI_RemoveMarketplace_RunsExactCommand(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin marketplace remove kit": {Stdout: "removed\n", ExitCode: 0},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.RemoveMarketplace(context.Background(), "kit"); err != nil {
		t.Fatalf("RemoveMarketplace() error = %v", err)
	}
	if len(r.Calls) != 1 {
		t.Fatalf("calls = %+v, want exactly one", r.Calls)
	}
	got := r.Calls[0].Name + " " + strings.Join(r.Calls[0].Args, " ")
	if got != "claude plugin marketplace remove kit" {
		t.Errorf("command = %q", got)
	}
}

func TestPluginCLI_RemoveMarketplace_TolerantOfNotFound(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin marketplace remove kit": {Stderr: "✘ Failed to remove marketplace: Marketplace 'kit' not found\n", ExitCode: 1},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.RemoveMarketplace(context.Background(), "kit"); err != nil {
		t.Fatalf("RemoveMarketplace() error = %v, want nil (tolerant of 'not found')", err)
	}
}

func TestPluginCLI_RemoveMarketplace_OtherFailureErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin marketplace remove kit": {Stderr: "boom\n", ExitCode: 1},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	_, err := cli.RemoveMarketplace(context.Background(), "kit")
	if err == nil {
		t.Fatal("RemoveMarketplace() error = nil, want an error")
	}
	if !strings.Contains(err.Error(), "boom") {
		t.Errorf("error = %v, want it to mention the failure output", err)
	}
}

func TestPluginCLI_RemoveMarketplace_UnrelatedNotFoundErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Responses: map[string]runnertest.Response{
			"claude plugin marketplace remove kit": {Stderr: "Error: settings file not found\n", ExitCode: 1},
		},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.RemoveMarketplace(context.Background(), "kit"); err == nil {
		t.Fatal("RemoveMarketplace() error = nil, want an error for a 'not found' that is not the marketplace itself")
	}
}

func TestPluginCLI_RemoveMarketplace_LaunchFailureErrors(t *testing.T) {
	r := &runnertest.FakeRunner{
		Default: runnertest.Response{Err: context.DeadlineExceeded},
	}
	cli := claudeplugin.PluginCLI{Runner: r}

	if _, err := cli.RemoveMarketplace(context.Background(), "kit"); err == nil {
		t.Fatal("RemoveMarketplace() error = nil, want an error on launch failure")
	}
}
