# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.2.4]

### Added
- **Dynamic Template Input**: The `call` command now supports passing templates directly via `--template-json` and `--template-base64` flags, enabling more flexible scripting and automation.
- **Stdin Support for Variables**: The `call` command can now read variable content from standard input (stdin) using the format `name:text:-` or `name:file:-`.

### Changed
- **Breaking Change**: The `--var` flag syntax for the `call` command has been changed. The format is now consistently `name:value` (shorthand for `name:text:value`) or `name:type:value`.
- The `file` variable type now reads file content as a raw string, without any special encoding.

### Removed
- Removed `base64` as a supported variable type for the `--var` flag. Base64 encoded templates should be used with the `--template-base64` flag instead.
