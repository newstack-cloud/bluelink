# Changelog

## [0.4.4](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.4.3...deploy-engine-client/v0.4.4) (2026-04-28)


### Dependencies

* **deploy-engine-client:** bump up blueprint lib to 0.45.0 ([67053a9](https://github.com/newstack-cloud/bluelink/commit/67053a97e5973b639acf0c82f423b39ba589292c))

## [0.4.3](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.4.2...deploy-engine-client/v0.4.3) (2026-04-26)


### Dependencies

* **deploy-engine-client:** bump up blueprint state to 0.8.2 ([e347859](https://github.com/newstack-cloud/bluelink/commit/e3478599375c70162fb17414349c01b0193083ca))

## [0.4.2](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.4.1...deploy-engine-client/v0.4.2) (2026-04-25)


### Dependencies

* **deploy-engine-client:** bump up blueprint state to 0.8.1 ([fb7755d](https://github.com/newstack-cloud/bluelink/commit/fb7755db2e1a8eb94f8816809832704ab6c29797))

## [0.4.1](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.4.0...deploy-engine-client/v0.4.1) (2026-04-25)


### Dependencies

* **deploy-engine-client:** bump up bluelink libs ([842665d](https://github.com/newstack-cloud/bluelink/commit/842665d0e081562798487651f89357ca0088d8b4))

## [0.4.0](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.3.1...deploy-engine-client/v0.4.0) (2026-04-20)


### Features

* **deploy-engine-client:** add support for removal policy and retained resources ([ddccb4e](https://github.com/newstack-cloud/bluelink/commit/ddccb4e74bccae4c57abdcd4eb8e37ad85b65ea2))

## [0.3.1](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.3.0...deploy-engine-client/v0.3.1) (2026-04-19)


### Dependencies

* **deploy-engine-client:** bump blueprint and blueprint state libs ([3c5b34e](https://github.com/newstack-cloud/bluelink/commit/3c5b34e6c52e5ecda15d863a134c73b11fd058b5))
* **deploy-engine-client:** bump the go-deps group ([6562aae](https://github.com/newstack-cloud/bluelink/commit/6562aae6ccc0d1acd28165df82658758fb4865b4))

## [0.3.0](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.2.0...deploy-engine-client/v0.3.0) (2026-01-11)


### Features

* **deploy-engine-client:** add cleanup operation tracking functionality ([86afcff](https://github.com/newstack-cloud/bluelink/commit/86afcffb6ea6f8963473e69696e7916b6bf2bfa4))


### Dependencies

* **deploy-engine-client:** update blueprint core and state libs ([8e6452c](https://github.com/newstack-cloud/bluelink/commit/8e6452c3399b5b86c12a54d4c61350626be59702))

## [0.2.0](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.1.2...deploy-engine-client/v0.2.0) (2026-01-06)


### Features

* **deploy-engine-client:** add behaviour to stream from an offset event id returned in responses ([98706b9](https://github.com/newstack-cloud/bluelink/commit/98706b953a9da2f87f2b0dd001b1b0f874013191))
* **deploy-engine-client:** add full support for deploy auto-rollback behaviour ([1e94a4f](https://github.com/newstack-cloud/bluelink/commit/1e94a4fea3216e747a20c51c5fe6ea3e4025ed62))
* **deploy-engine-client:** add functionality to list blueprint instances ([aa2b3e9](https://github.com/newstack-cloud/bluelink/commit/aa2b3e9a3a5bcf2597eac258d8ada9ec53442efa))
* **deploy-engine-client:** add helpers for drift blocked responses ([d35a118](https://github.com/newstack-cloud/bluelink/commit/d35a118f42113edd82947a0da092038e708a16c2))
* **deploy-engine-client:** add new error for trying to use deploy changeset for destroy ([fe17f53](https://github.com/newstack-cloud/bluelink/commit/fe17f53be0efa8d2b4f6214093f559caea5a8bb6))
* **deploy-engine-client:** add new options to stage, deploy and destroy types ([8e5fed5](https://github.com/newstack-cloud/bluelink/commit/8e5fed5e99cf1893f1e4a3d5b8a48f09d0808cbb))
* **deploy-engine-client:** add support for drift detection and reconciliation for child blueprints ([cccbbad](https://github.com/newstack-cloud/bluelink/commit/cccbbadd7af7e0c3db620a288134a11fe38bfe06))
* **deploy-engine-client:** add support for drift/reconciliation endpoints ([41cd99f](https://github.com/newstack-cloud/bluelink/commit/41cd99f592764f696b1e0800e0501c619ec87bf7))
* **deploy-engine-client:** update types to support system tagging configuration ([f44e62e](https://github.com/newstack-cloud/bluelink/commit/f44e62e3f3eebe3b88233800d1abefc70ec0e555))


### Bug Fixes

* **deploy-engine-client:** add fix for accessing event from closed channel ([af2ecb4](https://github.com/newstack-cloud/bluelink/commit/af2ecb4c2df2872ef20839f37317eac616aa0f77))
* **deploy-engine-client:** add fix for error message summary ([4f469be](https://github.com/newstack-cloud/bluelink/commit/4f469be15dac21d20703304d6b3740f1a1223bbf))


### Dependencies

* **deploy-engine-client:** update blueprint and blueprint state libs ([a4c3ba8](https://github.com/newstack-cloud/bluelink/commit/a4c3ba8b4adbbbadf88c2c99e7226a02507e7001))
* **deploy-engine-client:** update blueprint core and state libs ([30fae32](https://github.com/newstack-cloud/bluelink/commit/30fae32269612b2020708d87706a784f66702801))

## [0.1.2](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.1.1...deploy-engine-client/v0.1.2) (2025-12-12)


### Bug Fixes

* **deploy-engine-client:** add missing instance name for creating blueprint instances ([cdb64a3](https://github.com/newstack-cloud/bluelink/commit/cdb64a3781d03ae2a90c7526d1c6e24cc7a8c718))

## [0.1.1](https://github.com/newstack-cloud/bluelink/compare/deploy-engine-client/v0.1.0...deploy-engine-client/v0.1.1) (2025-12-10)


### Bug Fixes

* **deploy-engine-client:** correct event type for the change staging complete event ([834e105](https://github.com/newstack-cloud/bluelink/commit/834e1056e2a2e1b0e7e93a9f67b7b16ab0a4df10))


### Dependencies

* **deploy-engine-client:** bump the go-deps group ([f1d55d6](https://github.com/newstack-cloud/bluelink/commit/f1d55d682d16e8c3b389f30a99547fdfb1b5f9e3))

## Changelog

All notable changes to this project will be documented in this file.
