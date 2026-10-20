# Changelog

All notable changes to this project are documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Changed

- `s2t upgrade` now compares against the latest GitHub release, asks to confirm
  (`--yes` skips the prompt), then downloads the checksum-verified binary for
  this OS and architecture and replaces the running one. Homebrew installs
  still run `brew upgrade`.

### Added

- `s2t version` prints the version and the OS/arch release asset `upgrade` would download.

## [0.2.2] - 2026-10-20

### Added

- `s2t upgrade` runs `brew upgrade alialjaffer/tap/s2t` for Homebrew installs.

### Fixed

- `s2t set` and `s2t edit` now tab-complete resource names and `--namespace`.

## [0.2.0] - 2026-10-09

### Added

- `s2t set <name> [key=value ...]` upserts plaintext keys into a live Secret or
  ConfigMap. It merges `key=value` arguments and/or `-f`/stdin (JSON, YAML, or
  env), flattens nested JSON/YAML with `__`, prints a `kubectl patch`-ready
  payload by default, and applies it with `--apply` (`--dry-run` validates
  server-side first). Values never need hand-encoding to base64.
- `s2t edit <name> -n <ns>` opens a live resource's data keys (and nothing
  else) in `$EDITOR`, diffs them, and shows a merge patch to confirm before
  applying. Added and updated keys go through `stringData`; deleted keys are
  removed with a `null` in `data`.
- Homebrew install, via `brew install alialjaffer/tap/s2t`.

## [0.1.0] - 2026-09-06

### Added

- Initial release. Decode Kubernetes Secrets and ConfigMaps from a file, stdin,
  or a live `kubectl` fetch, with `plain`, `env`, `json`, `jsonc`, and `yaml`
  output, `--only` and `--mask` filters, key-by-key `s2t diff` across files or
  namespaces, and client-side Sealed Secrets decryption.

[Unreleased]: https://github.com/alialjaffer/s2t/compare/v0.2.0...HEAD
[0.2.0]: https://github.com/alialjaffer/s2t/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/alialjaffer/s2t/releases/tag/v0.1.0
