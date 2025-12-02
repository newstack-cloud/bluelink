# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.3](https://github.com/newstack-cloud/bluelink/compare/common/v0.3.2...common/v0.3.3) (2025-12-02)


### Features

* **common:** add clock and refactor test suite to testify ([abc2688](https://github.com/newstack-cloud/bluelink/commit/abc268859a5d44b4b7188639b03bd0b92092e0d0))
* **common:** add helper for snapshots with safe file names ([3311f67](https://github.com/newstack-cloud/bluelink/commit/3311f67d5d415e67a3c17e364629994af14a52cd))
* **common:** add signature v1 implementation for go ([407ee4a](https://github.com/newstack-cloud/bluelink/commit/407ee4a2bfbebd2051c7a34f50acc3a66f038a3d))


### Bug Fixes

* **common:** update celerity signature to bluelink signature ([39c50a9](https://github.com/newstack-cloud/bluelink/commit/39c50a92b864d13a8cd726086b9a8f286f939a20))

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
