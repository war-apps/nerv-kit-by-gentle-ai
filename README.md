# nerv-kit

nerv-kit is the shared library of the NERV ecosystem CLIs ([nerv](https://github.com/war-apps/nerv-by-gentle-ai), nerv-task and nerv-release): the Claude Code plugin registration, the self-upgrade machinery and the ai-workflow configuration layout they all need, with every nerv-specific name turned into a parameter.

## Packages

### `runner`

The sole seam onto the real machine: running external commands, resolving the home directory, and looking up executables on `PATH`. `runner/runnertest` provides `FakeRunner`, a scripted test double, so no other package needs to launch a real process in its tests.

```go
var r runner.Runner = runner.ExecRunner{}
stdout, stderr, exitCode, err := r.Run(ctx, "claude", "--version")
```

### `atomicfile`

Writes a file's new content without ever leaving a reader to observe a partial write: a temp file in the same directory, then a rename, with an optional backup of any existing file and an optional verification pass.

```go
backup, err := atomicfile.Save(path, data, atomicfile.Options{
    Now:          time.Now(),
    BackupSuffix: "bak-mytool-",
})
```

### `claudeplugin`

Materializes a plugin tree onto disk, registers it in Claude Code as a directory marketplace, drives the `claude plugin` CLI, and edits Claude Code's own `settings.json` and `installed_plugins.json`.

```go
result, err := claudeplugin.Materialize(pluginFS, destDir, claudeplugin.Marketplace{
    Name:      "my-tool",
    OwnerName: "My Org",
})
```

### `selfupgrade`

Replaces the running binary with a newer GitHub release: resolves the newest release of a channel, downloads the platform's archive and checksums, verifies the SHA-256, and installs it over the running binary. `selfupgrade/selfupgradetest` provides a fake GitHub releases API and download server, parameterized by binary name, for testing without a real network call.

```go
client := selfupgrade.Client{APIBase: apiBase, DownloadBase: downloadBase, BinaryName: "my-tool"}
tag, err := client.Resolve(ctx, selfupgrade.Stable)
```

### `aiworkflow`

Resolves the user-scope and project-scope configuration locations the NERV ecosystem CLIs share, plus a generic YAML load/save of a Go struct over `atomicfile`.

```go
path := aiworkflow.UserConfigPath(home, runtime.GOOS, "my-tool")
var cfg MyConfig
found, err := aiworkflow.Load(path, &cfg)
```

## Versioning

nerv-kit releases `v0.x` tags until the three consumer CLIs (nerv, nerv-task, nerv-release) depend on it; expect breaking changes between minor versions during this period.

## Development

```sh
go build ./...
go vet ./...
go test ./...
gofmt -l .   # must print nothing
```

Packages do not import each other except `claudeplugin` → `runner`, `atomicfile` and `aiworkflow` → `atomicfile`.
