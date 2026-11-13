# nerv-kit: the shared library of the NERV ecosystem CLIs

Objective: extract from nerv the code that nerv, nerv-task and nerv-release all need (Claude Code plugin registration, self-upgrade, ai-workflow configuration), with nerv-specific names turned into parameters.
Master plan: `war-apps/nerv-by-gentle-ai` → `odd/tasks/ecosystem-split.md`, phase P1 (specs S3, S4, S8).
Branch: `feat/initial-kit` from `develop`. Module `github.com/war-apps/nerv-kit-by-gentle-ai`, Go 1.26. Releases: `v0.x` tags until the three consumers use it. TDD: strict (session configuration); runner `go test ./...`.

## Specs

- **K1** Package `runner` (+ `runnertest`): the `Runner` interface, `ExecRunner`, `HomeDir`, `LookPath`, and the `FakeRunner` test double, ported from nerv `internal/env` and `internal/env/envtest`.
- **K2** Package `atomicfile`: `Save` (with backup) and `Write`, ported from nerv `internal/atomicfile`; tests assert file modes only where the platform has POSIX modes.
- **K3** Package `claudeplugin`: `Materialize`/`MaterializeBundle` with a `Marketplace{Name, OwnerName}` parameter; `PluginCLI` (add/remove marketplace, install/uninstall); `Settings` (load, register marketplace, enable/remove plugin, save with a caller-supplied backup suffix); `InstalledPlugin(path, id)`; `CacheAgentsDir(home, marketplace, plugin, version)`; `PluginVersion(fs.FS)`. Ported from nerv `internal/plugin/materialize.go`, `internal/claude`, `internal/version`. No literal "nerv".
- **K4** Package `selfupgrade`: `Client{HTTP, APIBase, DownloadBase, BinaryName}` with `Resolve(channel)` and `Download`; `VerifyChecksum`; `ArchiveName(binary, tag, goos, goarch)` (`.zip` on Windows); `Replace(archive, exe, goos, binary)` including the Windows rename-aside and restore; `CheckPlatform(goos, goarch, binary, installHint)`; `CleanupStaleReplacement`; `IsNewer` with a small embedded semver comparator. Ported from nerv `internal/upgrade` and the `Valid`/`Compare` slice of `internal/versioning/semver.go`. Includes a reusable `httptest` GitHub-releases fake for consumers.
- **K5** Package `aiworkflow` (new): `UserConfigDir(home, goos)` (XDG on Linux, `%AppData%` on Windows, macOS per nerv `internal/paths`), `UserConfigPath(home, goos, tool)`, `ProjectDir(root)`, `ProjectConfigPath(root, tool)`, and struct-based `Load`/`Save` over `gopkg.in/yaml.v3` + `atomicfile` (user files 0600).
- **K6** CI: `gofmt -l`, `go vet ./...`, `go test ./...` on Ubuntu and Windows for pull requests to `develop` and `main`. README with the packages and their purpose.
- **K7** Packages do not import each other except `claudeplugin`/`aiworkflow` → `atomicfile`, `claudeplugin` → `runner`. No nerv-only code (config editor, extensions, gentle-ai, workflow sections).

## Tasks

| ID | Specs | Route | Status | Commit |
|---|---|---|---|---|
| T1 | K1-K7 | delegated writer | done: RED observed per package, GREEN; gofmt/vet/build clean; linux vet and windows build cross-checked; parent reran `go test -count=1 ./...` (all ok) | f8a791b, 7cca3ad, 4f091e7, 4cf24b6, 9317af2, 31e3fb2, 8c3b94b |

## Log

- **L1** (2026-11-13) API approved by the user ("ok") from the P1 mapping (nerv master doc L4).
- **L2** (2026-11-13) Implemented in 7 commits. Deviations: `MaterializeBundle` lists extension folders by their `plugin.json` only (nerv's provider/bundle graph stays in nerv); `InstalledPlugins` wrapper and nerv URL constants dropped (callers pass `APIBase`, `DownloadBase`, install hint); `PlatformError` instead of a refusal package; `aiworkflow.Save` chmods after `atomicfile.Save` to honor a non-default mode.
- **L3** (2026-11-13) Security review finding fixed (4cf24b6): backups were written 0644 regardless of the source (a 0600 user config backup became world-readable); now the backup keeps the source mode (0600 fallback) and is created with O_EXCL. The same bug exists in nerv `internal/atomicfile` (follow-up there).
