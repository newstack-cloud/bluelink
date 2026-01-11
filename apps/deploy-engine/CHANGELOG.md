# Changelog

## [0.3.0](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.2.0...deploy-engine/v0.3.0) (2026-01-11)


### Features

* **deploy-engine:** add support for cleanup operation tracking ([daf62cc](https://github.com/newstack-cloud/bluelink/commit/daf62cc98e3ca4851583a2bd9fa7750832e92c7c))


### Dependencies

* **deploy-engine:** update blueprint core and state libs ([5037b95](https://github.com/newstack-cloud/bluelink/commit/5037b9583b909aa49ec4ebde5e5c3938145e3e08))

## [0.2.0](https://github.com/newstack-cloud/bluelink/compare/deploy-engine/v0.1.5...deploy-engine/v0.2.0) (2026-01-06)


### Features

* **blueprint-state:** add persistence for reconciliation results ([9f16d59](https://github.com/newstack-cloud/bluelink/commit/9f16d59772aab7f09a3b109042684aa6d89d0015))
* **blueprint:** add support for advanced reconciliation and drift checks ([b889111](https://github.com/newstack-cloud/bluelink/commit/b8891116050e5f2c5d11d48e37853cb65e6f28be))
* **deploy-engine:** add endpoint for listing blueprint instances ([3e5a94c](https://github.com/newstack-cloud/bluelink/commit/3e5a94c47453cf38d0ccf8dce45c2c4616439f08))
* **deploy-engine:** add endpoint to clean up reconciliation results ([8fd754a](https://github.com/newstack-cloud/bluelink/commit/8fd754a4182cc0bfd289e13bb05b65605e4b6b32))
* **deploy-engine:** add fully functioning auto-rollback behaviour for deployments ([0c798e1](https://github.com/newstack-cloud/bluelink/commit/0c798e1cebc500537fdb3f45135045002b5694ff))
* **deploy-engine:** add instance not found check that produces error with instance name ([6c6c3c3](https://github.com/newstack-cloud/bluelink/commit/6c6c3c3d59379ce6b16530b108875e0e3479951d))
* **deploy-engine:** add last event id to responses to be used as offsets for streaming ([910b413](https://github.com/newstack-cloud/bluelink/commit/910b41385ab0746f13207f088ca99d16010b0f39))
* **deploy-engine:** add support for a configurable deployment drain timeout ([a4b8921](https://github.com/newstack-cloud/bluelink/commit/a4b8921b7169540ff6e246aed1dd523be1587c98))
* **deploy-engine:** add support for advanced drift check and reconciliation ([4e12386](https://github.com/newstack-cloud/bluelink/commit/4e12386271c84a7c6cde84a8eb7df2ebbd88a424))
* **deploy-engine:** add support for drift detection and reconciliation for child blueprints ([c8213fe](https://github.com/newstack-cloud/bluelink/commit/c8213fe898a577ad9723f55a0448937d6443f766))
* **deploy-engine:** add support for new auto rollback and force flags ([061e7b2](https://github.com/newstack-cloud/bluelink/commit/061e7b243a293879a9c74e49410ac14e0c23221e))
* **deploy-engine:** add support for system-level tagging and resource system metadata ([4bea385](https://github.com/newstack-cloud/bluelink/commit/4bea38576499fb28555152b40eebfdac2e98a4b6))
* **deploy-engine:** add support for tagging config to enable provenance tagging in provider plugins ([aa28a04](https://github.com/newstack-cloud/bluelink/commit/aa28a04c1551f601a420f18d4fcda9136abc84c0))


### Bug Fixes

* **deploy-engine:** add correction to handle instance name provided for new deployment ([a156e07](https://github.com/newstack-cloud/bluelink/commit/a156e073bc5a92d3d1d174ae72ac3e94c8598f93))
* **deploy-engine:** add fixes for the destroy endpoint and event streams ([a4ab692](https://github.com/newstack-cloud/bluelink/commit/a4ab692592aa088dc6ecae544ecd3dfca51d567b))


### Dependencies

* **deploy-engine:** update bluelink lib dependencies ([563c3ac](https://github.com/newstack-cloud/bluelink/commit/563c3ac410e8fe958c638a3433958f43838bce61))
* **deploy-engine:** update blueprint and blueprint state libs ([2733329](https://github.com/newstack-cloud/bluelink/commit/2733329ff183d0d1119af7f0b28f1a75ad4166d4))
* **deploy-engine:** update dependencies ([92593dc](https://github.com/newstack-cloud/bluelink/commit/92593dc5036800d077598fc15fd67cb1ebdc314b))

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
