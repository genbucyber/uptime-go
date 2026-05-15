# Changelog

All notable changes to Uptime Go will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- Package app with Nix (`flake.nix`, `flake.lock`)

### Changed
- Improve timeout error messages for clearer diagnostics
- Refactor API: separate report endpoints

## [0.4.3] - 2026-02-11

### Fixed
- Update repository owner for self-update mechanism

## [0.4.2] - 2026-02-11

### Changed
- Improve incident notification messages
- Change user agent string
- Do not exit if agent configuration does not exist

## [0.4.1] - 2026-01-31

### Added
- IP type configuration for monitors (IPv4 / IPv6)

## [0.4.0] - 2026-01-31

### Added
- Retry configuration with per-monitor granular timeouts
- Follow redirects option for website checks
- Additional configuration fields for monitoring parameters (runtime values)

### Changed
- Treat redirect responses as successful checks
- Enhance monitor tests with new incident scenarios and status determination

## [0.3.0] - 2025-11-30

### Added
- Self-update functionality (`--self-update`)
- Release workflow automation

## [0.2.3] - 2025-11-30

### Fixed
- Error message formatting

### Changed
- Release workflow improvements

## [0.2.2] - 2025-11-13

### Fixed
- Incident event hardcoded values — improved incident readability

### Changed
- Update tag workflow

## [0.2.1] - 2025-11-13

### Changed
- Update user agent string

## [0.2.0] - 2025-11-12

### Fixed
- Handle empty configuration gracefully

## [0.1.3] - 2025-11-11

### Changed
- Update CI test workflow

## [0.1.2] - 2025-08-27

### Added
- IP address caching
- HTTP API server with structured logging (zerolog)
- Nix flake for development environment
- Air hot reload for local development

### Changed
- Replace standard library log with zerolog
- Remove `setConfig` and `report` subcommands
- Simplify configuration handling
- Update incident payload format

### Fixed
- Default monitor startup
- Test database initialization

## [0.1.1] - 2025-08-23

### Added
- Initial release — uptime monitoring core
- HTTP(S) endpoint monitoring with configurable intervals
- Webhook notifications for incidents
- Response time tracking and certificate expiry checks
- Historical data storage with SQLite
- CI test action

### Fixed
- Prevent panic during certificate validation
- Improve certificate handling edge cases
