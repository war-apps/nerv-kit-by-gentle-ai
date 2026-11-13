# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.1] - 2026-11-13

### Fixed
- atomicfile: a second save in the same second no longer fails on its backup; the backup takes the next free name (`-1`, `-2`, ...) and still never writes through an existing file
- atomicfile: a verification failure after the rename now restores the original content (the restore was refused by the O_EXCL backup copy and the error was ignored)

## [0.1.0] - 2026-11-13

First release: the shared Go library of the NERV ecosystem CLIs, extracted
from nerv with every nerv-specific name turned into a parameter.

### Added
- runner: the command `Runner` seam and its `FakeRunner` test double
- atomicfile: atomic writes with backups that keep the source permissions
- claudeplugin: plugin marketplace materialization, `claude plugin` CLI, `settings.json` editing, installed-plugin lookup
- selfupgrade: GitHub release self-upgrade (channels, `.tar.gz`/`.zip`, checksums, Windows rename-aside) and a fake releases server for tests
- aiworkflow: `ai-workflow/<tool>.yaml` user and project paths, struct-based YAML load and save
