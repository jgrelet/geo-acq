# Changelog

All notable changes to this project will be documented in this file.

The format is based on Keep a Changelog, and this project follows semantic versioning.

## [Unreleased]

### Added

- Display the application version in the Wails window title.
- Remember the last successfully loaded TOML configuration file in the user configuration directory and reload it on startup when it still exists.
- Ignore SQLite WAL and shared-memory sidecar files in Git.
- Support optional custom SQLite backup paths in `[backup]` with `raw_path` and `processed_path`.
- Treat backup paths that point to directories as output directories for generated SQLite file names.

### Changed

- Checkpoint and truncate SQLite WAL files when closing acquisition stores so stopped sessions are reflected in the main `.sqlite` files.
- Persist device simulation toggles by updating the loaded TOML `mode` value instead of keeping hidden runtime overrides.
- Inject the application version at build time so release tags appear in the GUI title.

## [0.1.0] - 2026-06-04

### Added

- Initial GUI acquisition prototype with TOML configuration loading and SQLite persistence.
