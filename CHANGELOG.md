# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project follows semantic versioning.

## [Unreleased]

### Added

- Display the application version in the Wails window title.
- Remember the last successfully loaded TOML configuration file in the user configuration directory and reload it on startup when it still exists.
- Ignore SQLite WAL and shared-memory sidecar files in Git.
- Support optional custom SQLite backup paths in `[backup]` with `raw_path` and `processed_path`.

### Changed

- Checkpoint and truncate SQLite WAL files when closing acquisition stores so stopped sessions are reflected in the main `.sqlite` files.

## [0.1.0] - 2026-06-04

### Added

- Initial GUI acquisition prototype with TOML configuration loading and SQLite persistence.
