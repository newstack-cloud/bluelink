# Changelog

## [0.4.1](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.4.0...plugin-framework/v0.4.1) (2026-02-01)


### Dependencies

* **plugin-framework:** bump the go-deps group across 1 directory with 2 updates ([39abecc](https://github.com/newstack-cloud/bluelink/commit/39abeccf204992e7c7e711787d9c5e998417c644))

## [0.4.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.3.0...plugin-framework/v0.4.0) (2026-01-27)


### Features

* **plugin-framework:** add layer to dynamically resolve linkable resource types ([ea1d9e3](https://github.com/newstack-cloud/bluelink/commit/ea1d9e3d7dc8bdbef02d891f37babbbbda053b8a))


### Dependencies

* **plugin-framework:** update blueprint and common libs ([67e8ed1](https://github.com/newstack-cloud/bluelink/commit/67e8ed14d239058f9ae577cb849d05b6f35c9ebb))

## [0.3.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.2.0...plugin-framework/v0.3.0) (2026-01-27)


### Features

* **blueprint:** add field to more accurately map annotation to resource ([478e984](https://github.com/newstack-cloud/bluelink/commit/478e9841dad47d3a0d2d1b80dc3a1b25baa40142))
* **plugin-framework:** add applies to field to link annotation definition protobuf ([1262bf7](https://github.com/newstack-cloud/bluelink/commit/1262bf73d73b82bd08aa455b7f9fe6e7e8a00739))
* **plugin-framework:** add support for killing plugin processes ([a00c202](https://github.com/newstack-cloud/bluelink/commit/a00c2024e903a90191c21d7b474f3b04fac8b874))


### Bug Fixes

* **plugin-framework:** add defensive checks for incomplete inputs ([70484a8](https://github.com/newstack-cloud/bluelink/commit/70484a898fd8aeae9163c835db26e96d2d382608))
* **plugin-framework:** add defensive checks to support rapid editing ([84ad6ac](https://github.com/newstack-cloud/bluelink/commit/84ad6ac71fe2f31f1072a99f1641da53c8ebf94e))
* **plugin-framework:** add missing resource schema fields to protobuf serialisation ([1fb4494](https://github.com/newstack-cloud/bluelink/commit/1fb4494282c62f2df4cc84cc3acc0dac03be0f9d))


### Dependencies

* **plugin-framework:** update blueprint core lib version ([2557bc3](https://github.com/newstack-cloud/bluelink/commit/2557bc37ae50090e91d180c8202ca19c5ea34cb5))

## [0.2.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.1.4...plugin-framework/v0.2.0) (2026-01-06)


### Features

* **plugin-framework:** add helpers for looking up resources by tags ([135bdbd](https://github.com/newstack-cloud/bluelink/commit/135bdbda26d2d51887f2455fd0c829604fc1dfeb))
* **plugin-framework:** add support for retrieving link intermediary resource external state ([d2be290](https://github.com/newstack-cloud/bluelink/commit/d2be2904ee6ba1fae5383d34e3f56175262d3607))
* **plugin-framework:** add support for sorting arrays in resource schema by a field for comparisons ([176269c](https://github.com/newstack-cloud/bluelink/commit/176269ce0fec3bd1cdb2a7b4e241fe6dc60dfe47))
* **plugin-framework:** add support for system metadata for system-level tagging ([3c71f74](https://github.com/newstack-cloud/bluelink/commit/3c71f7415fe34540dee0e5bfdda22856817b29f2))


### Bug Fixes

* **plugin-framework:** add missing resource name for get external state call ([d17675d](https://github.com/newstack-cloud/bluelink/commit/d17675d182b4a456f4595a94a8434ba258d8e93b))


### Dependencies

* **plugin-framework:** update blueprint core lib ([9808f08](https://github.com/newstack-cloud/bluelink/commit/9808f08976f914f5cad405c3f2c64986f47a1f73))

## [0.1.4](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.1.3...plugin-framework/v0.1.4) (2025-12-16)


### Bug Fixes

* **plugin-framework:** allow empty resource spec states to reveal more useful errors ([9b51614](https://github.com/newstack-cloud/bluelink/commit/9b516148c3b84129bfbca35bdf0111bfb5d5adec))


### Dependencies

* **plugin-framework:** update blueprint lib to 0.36.4 ([87c8ec5](https://github.com/newstack-cloud/bluelink/commit/87c8ec51ed4a49b1233b116eb2643602babd9b41))

## [0.1.3](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.1.2...plugin-framework/v0.1.3) (2025-12-09)


### Bug Fixes

* **plugin-framework:** fix broken support for relative paths in plugin path ([9987f72](https://github.com/newstack-cloud/bluelink/commit/9987f72b59490648ce28011d398714c2ca9813c0))


### Dependencies

* **plugin-framework:** update core blueprint lib to 0.36.1 with link context fix ([1accff2](https://github.com/newstack-cloud/bluelink/commit/1accff2db637d8720eb0b8dc9b4c1d0f2a418870))

## [0.1.2](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.1.1...plugin-framework/v0.1.2) (2025-12-06)


### Bug Fixes

* **plugin-framework:** prepare plugin framework library for indexing in the go registry ([e89bba3](https://github.com/newstack-cloud/bluelink/commit/e89bba39e17fe4e5467813a5eb100e9ac95baafc))

## [0.1.1](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.1.0...plugin-framework/v0.1.1) (2025-12-06)


### Bug Fixes

* **plugin-framework:** update path list separator for plugin path to support windows ([681f5e2](https://github.com/newstack-cloud/bluelink/commit/681f5e22de3365e8bce7d17d041fb889d40fd1fa))


### Dependencies

* **plugin-framework:** bump the go-deps group ([b1a8461](https://github.com/newstack-cloud/bluelink/commit/b1a8461b482b6407bc14979a436e08f2fca5a718))
