# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added
- LLM-based validation for reducing false positives using local models
- Support for LM Studio integration with OpenAI-compatible API
- Context-aware PI validation with configurable risk assessment
- Enhanced validation algorithms for Australian PI types:
  - TFN validation with proper modulo 11 checksum
  - Medicare number validation with IRN position checking
  - BSB validation against bank code ranges
  - Driver license validation with state-specific formats
  - Phone number validation using Google's libphonenumber
- Comprehensive address validation for Australian postcodes
- LLM configuration options via CLI flags and config files
- Documentation for LLM integration setup and usage

### Changed
- Improved PI detection accuracy with algorithmic validation
- Enhanced context extraction for better LLM analysis
- Updated detection confidence scoring system

### Fixed
- Fixed compilation error with missing isValidAustralianAddress function
- Corrected TFN validation algorithm
- Fixed Medicare number validation logic
- Improved BSB and driver license validation accuracy

## [1.1.0] - 2024-06-19

### Added
- Multi-platform binary releases (Darwin, Linux, Windows)
- ARM64 support for Apple Silicon and Linux ARM
- Distribution verification scripts
- Enhanced release documentation

### Changed
- Improved release build process
- Updated binary naming conventions

## [1.0.0] - 2024-06-19

### Added
- Initial release of GitHub PI Scanner
- Australian PI detection for TFN, ABN, ACN, Medicare, BSB, driver licenses
- Context-aware pattern matching with confidence scoring
- Concurrent processing with worker pools
- Multiple report formats (JSON, CSV, SARIF, HTML)
- GitHub repository integration
- Docker support
- Comprehensive test coverage
- Security audit compliance

[Unreleased]: https://github.com/MacAttak/pi-scanner/compare/v1.1.0...HEAD
[1.1.0]: https://github.com/MacAttak/pi-scanner/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/MacAttak/pi-scanner/releases/tag/v1.0.0
