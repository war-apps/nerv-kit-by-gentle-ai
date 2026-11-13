# Changelog

All notable changes to this project are documented in this file.
The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [0.1.0] - 2026-11-13

First release: the shared Go library of the NERV ecosystem CLIs, extracted
from nerv with every nerv-specific name turned into a parameter.

### Added
- runner: the command `Runner` seam and its `FakeRunner` test double
- atomicfile: atomic writes with backups that keep the source permissions
- claudeplugin: plugin marketplace materialization, `claude plugin` CLI, `settings.json` editing, installed-plugin lookup
- selfupgrade: GitHub release self-upgrade (channels, `.tar.gz`/`.zip`, checksums, Windows rename-aside) and a fake releases server for tests
- aiworkflow: `ai-workflow/<tool>.yaml` user and project paths, struct-based YAML load and save
