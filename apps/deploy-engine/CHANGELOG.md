# Changelog

## [0.1.5](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.1.4...deploy-engine/v0.1.5) (2025-12-12)


### Bug Fixes

* **deploy-engine:** add missing support for lookup by name and required name for instance creation ([505a1d4](https://github.com/newstack-cloud/bluelink/commit/505a1d4ae5edd3c0fed209b728e3354c465ad6d3))
* **deploy-engine:** add support for resolving relative paths for child blueprints ([f9bba1c](https://github.com/newstack-cloud/bluelink/commit/f9bba1c87ab8ae10719e3fe4c322fe6acaf8ef49))


### Dependencies

* **deploy-engine:** update blueprint and resolver libs to versions with fixes ([f76b512](https://github.com/newstack-cloud/bluelink/commit/f76b51237b4b4c7c352553fbe7bba474cdbc98f1))

## [0.1.4](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.1.3...deploy-engine/v0.1.4) (2025-12-06)


### Bug Fixes

* **deploy-engine:** correct plugin path to support windows path list separators ([68cf4b6](https://github.com/newstack-cloud/bluelink/commit/68cf4b60bfbdffd9d93db0728f8a0fa6b647374f))

## [0.1.3](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.1.2...deploy-engine/v0.1.3) (2025-12-06)


### Bug Fixes

* **deploy-engine:** make sure env vars are expanded in log file root directory ([62d979c](https://github.com/newstack-cloud/bluelink/commit/62d979cf9f8ac1bd9d4fc0ec88c4f0db46837748))

## [0.1.2](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.1.1...deploy-engine/v0.1.2) (2025-12-06)


### Bug Fixes

* **deploy-engine:** correct default paths on windows ([f58c89c](https://github.com/newstack-cloud/bluelink/commit/f58c89c3c0d0eb77f3b34afec0e7d9a41a6a9d6f))
* **deploy-engine:** correct paths for windows ([089f195](https://github.com/newstack-cloud/bluelink/commit/089f195ba8e12e553588cb4b4e4cba087484162c))


### Dependencies

* **deploy-engine:** bump the go-deps group ([881d14b](https://github.com/newstack-cloud/bluelink/commit/881d14bb1946b7c68b32a901d1ced417a6b23bba))
* **docker:** bump golang in /apps/deploy-engine ([d7e7db7](https://github.com/newstack-cloud/bluelink/commit/d7e7db7946dd370f6da79177d127c74e7bbed10a))

## [0.1.1](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.1.0...deploy-engine/v0.1.1) (2025-12-03)


### Features

* **deploy-engine:** add support for loading configuration from files ([77a2e3b](https://github.com/newstack-cloud/bluelink/commit/77a2e3bc873a0bc7db358759c6bd69c04ff0c78a))
* **deploy-engine:** make resolver s3 path style configurable ([2624775](https://github.com/newstack-cloud/bluelink/commit/2624775d4edf155099b389b975a2bc52466562eb))
