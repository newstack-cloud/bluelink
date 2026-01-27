# Changelog

All notable changes to this project will be documented in this file.

## [0.4.0](https://github.com/newstack-cloud/bluelink/compare/common/v0.3.2...common/v0.4.0) (2026-01-27)


### Features

* **common:** add utils for string similarities ([1a8593c](https://github.com/newstack-cloud/bluelink/commit/1a8593c52cf94febf22e2c87703b0813c37b82f3))


### Dependencies

* **common:** bump github.com/stretchr/testify ([37e7135](https://github.com/newstack-cloud/bluelink/commit/37e7135774f242bf6f4f777ff70ecac35de7daaa))

## [0.3.2] - 2025-06-24

### Fixed

- Corrects the go module path in the `go.mod` file to `github.com/newstack-cloud/bluelink/libs/common` for all future releases.

## [0.3.1] - 2025-06-10

### Fixed

- Corrects the go module path in the `go.mod` file to `github.com/newstack-cloud/celerity/libs/common` for all future releases.

## [0.3.0] - 2025-05-09

### Added

- Adds `testhelpers` package containing a helper function to create snapshot files with file names that can be used on windows systems. Using cupaloy/v2 out of the box will add "\*" to the file name for test suites with pointer receivers. This is not a valid character for file names on Windows systems. The helper function will remove the "\*" from the file name without every test suite across Celerity projects having to manually set the snapshot name.

## [0.2.0] - 2025-04-19

### Added

- Adds an implementation of the [Celerity Signature v1 specification](https://www.celerityframework.io/docs/auth/signature-v1) for Go.

## [0.1.0] - 2025-01-29

### Added

- Initial release of the library including utility functions for working with slices.
