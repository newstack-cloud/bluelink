# Changelog

## [0.17.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.16.0...plugin-framework/v0.17.0) (2026-09-05)


### ⚠ BREAKING CHANGES

* **plugin-framework:** provider plugins must serve UpdateLinkedResources in place of UpdateLinkResourceA and UpdateLinkResourceB, and the SDK's LinkDefinition takes UpdateLinkedResourcesFunc instead of the two per-side functions. LinkCompletionDurations field numbers have changed, so plugins must be rebuilt against this version.

### Features

* **plugin-framework:** add functionality to allow providers to release resource locks ([ae17b76](https://github.com/newstack-cloud/bluelink/commit/ae17b76604e006a9361f09e802ff87fba20f5807))
* **plugin-framework:** add resource role for grouping resources and enrich shared parents ([854889b](https://github.com/newstack-cloud/bluelink/commit/854889b0059461213b7e29599cbbf2bcc3944510))
* **plugin-framework:** add test helper for declarative link contributions ([a15e3c1](https://github.com/newstack-cloud/bluelink/commit/a15e3c1a334d49c107e8bad4b2505f277c27d82f))
* **plugin-framework:** carry a link's contribution declaration over the wire ([fcd20ea](https://github.com/newstack-cloud/bluelink/commit/fcd20ea643e54e38ac10453a81fe52a7732ddf14))
* **plugin-framework:** carry link field ownership over the protocol ([4b92e16](https://github.com/newstack-cloud/bluelink/commit/4b92e1668c434f7aa7ad6d71195ef111a65dbd70))
* **plugin-framework:** carry link modified resources over the protocol ([37c05de](https://github.com/newstack-cloud/bluelink/commit/37c05de19d356c087557871fcd7477404f2c0ffd))
* **plugin-framework:** carry the reason a stabilisation check was made ([b2b9fd8](https://github.com/newstack-cloud/bluelink/commit/b2b9fd836ef199795f177daad2678601a5d9f5d0))
* **plugin-framework:** carry unapplied link contributions over the wire ([74abadc](https://github.com/newstack-cloud/bluelink/commit/74abadc36bf31472c44591f7e689faf2b80dbc55))
* **plugin-framework:** replace the link resource A and B update RPCs ([ec12ed2](https://github.com/newstack-cloud/bluelink/commit/ec12ed298cbfbde4acdfb3423c494254b92fbed0))
* **plugin-framework:** serve ProduceResourceContributions for link plugins ([c5bab7b](https://github.com/newstack-cloud/bluelink/commit/c5bab7b952570a63d2e6176d20ea8339cbf28cbf))
* **plugin-framework:** tell a provider a deployment carries link contributions ([20d27b5](https://github.com/newstack-cloud/bluelink/commit/20d27b56f3ec76f6a5f0444a492086b51a9b8ed7))


### Bug Fixes

* **plugin-framework:** add corection to support multiple asserts for service mock calls ([6ba5ea0](https://github.com/newstack-cloud/bluelink/commit/6ba5ea055e8f3025587d0c5edf29d67a6325a6a9))
* **plugin-framework:** stop the test state container dropping fields on copy ([806cc5f](https://github.com/newstack-cloud/bluelink/commit/806cc5fb5e04a025ca6ea9a5a346114e1a719f57))


### Dependencies

* **plugin-framework:** bump blueprint lib to 0.52.0 ([d1e8d6d](https://github.com/newstack-cloud/bluelink/commit/d1e8d6d34a9f967681b724868f43db63ba2d44b9))
* **plugin-framework:** update plugin-framework go modules ([#277](https://github.com/newstack-cloud/bluelink/issues/277)) ([2f426dd](https://github.com/newstack-cloud/bluelink/commit/2f426ddbc127f411e8c6ff69cce07dbd70e9a4b5))

## [0.16.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.15.0...plugin-framework/v0.16.0) (2026-08-07)


### Features

* **plugin-framework:** add support for link capabilities ([75bb124](https://github.com/newstack-cloud/bluelink/commit/75bb124129d6f4d01bc124c2286e8ea029cde03d))


### Bug Fixes

* **plugin-framework:** pass declared link graph to transformers ([14f9ebd](https://github.com/newstack-cloud/bluelink/commit/14f9ebd145485419ab72d490af174b46cc603575))
* **plugin-framework:** report unsupported plugin item types as not found ([2bd07ef](https://github.com/newstack-cloud/bluelink/commit/2bd07ef24ea8842222eef99779f7f2a5d32fdf84))
* **plugin-framework:** tolerate nil transformer context in pb conversion ([3e758f6](https://github.com/newstack-cloud/bluelink/commit/3e758f6b0a0b66b0072ce8d518630b3dfe5c16d6))


### Dependencies

* **plugin-framework:** update module google.golang.org/grpc to v1.82.1 ([#260](https://github.com/newstack-cloud/bluelink/issues/260)) ([9eff48d](https://github.com/newstack-cloud/bluelink/commit/9eff48d7249a757f7d3606adec8fc4fe2cfdf5a2))
* **plugin-framework:** update module google.golang.org/grpc to v1.83.0 ([#268](https://github.com/newstack-cloud/bluelink/issues/268)) ([77fdb4b](https://github.com/newstack-cloud/bluelink/commit/77fdb4bba1d1ab0506287d7c32282907568b1a97))

## [0.15.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.14.0...plugin-framework/v0.15.0) (2026-07-19)


### Features

* **plugin-framework:** add support for computed when omitted fields ([13aa742](https://github.com/newstack-cloud/bluelink/commit/13aa7428aceba402524f4f873391f0b5b29cce4b))


### Dependencies

* **plugin-framework:** bump blueprint lib to 0.51.2 ([7260f6c](https://github.com/newstack-cloud/bluelink/commit/7260f6cf1081205206383f37ae82388eaffa1579))

## [0.14.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.13.0...plugin-framework/v0.14.0) (2026-06-25)


### Features

* **plugin-framework:** add support for carrying link activating resource fields ([ec54a49](https://github.com/newstack-cloud/bluelink/commit/ec54a49d923450bf12b7251b74fb0c4a212e2099))

## [0.13.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.12.0...plugin-framework/v0.13.0) (2026-06-23)


### Features

* **blueprint:** add support for capturing computed fields when stable ([e1d9098](https://github.com/newstack-cloud/bluelink/commit/e1d9098a7e3fedbce6a352642e96293bb5b6e6ec))
* **plugin-framework:** ensure computed fields are carried over grpc for stablised resources ([4449f88](https://github.com/newstack-cloud/bluelink/commit/4449f884ea71b8895b25b5142488ca58d0649de8))


### Dependencies

* **plugin-framework:** bump blueprint lib version to 0.49.0 ([0a033ef](https://github.com/newstack-cloud/bluelink/commit/0a033ef4eb19e727eeee4414ce42bcdc4ebb001a))

## [0.12.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.11.0...plugin-framework/v0.12.0) (2026-06-01)


### Features

* **plugin-framework:** add helpers for extracting annotations from bp schema ([4f772de](https://github.com/newstack-cloud/bluelink/commit/4f772def8fd42be39d322fa86d9598a54d027455))


### Dependencies

* **plugin-framework:** bump blueprint lib to 0.47.0 ([963dc1c](https://github.com/newstack-cloud/bluelink/commit/963dc1c9bf79d236156f850acdb67b36a0aed395))

## [0.11.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.10.0...plugin-framework/v0.11.0) (2026-05-13)


### Features

* **plugin-framework:** add support for threading values in a run context through transform pipeline ([a99405f](https://github.com/newstack-cloud/bluelink/commit/a99405f7548dcaaa7429e31dc7b482956a461a62))

## [0.10.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.9.0...plugin-framework/v0.10.0) (2026-05-10)


### Features

* **plugin-framework:** thread go context through transform pipeline ([ad67837](https://github.com/newstack-cloud/bluelink/commit/ad67837d0b2a9795c826db37ce9ad0079ad73db0))

## [0.9.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.8.0...plugin-framework/v0.9.0) (2026-05-09)


### Features

* **plugin-framework:** add helper to check for validation context ([76a23e9](https://github.com/newstack-cloud/bluelink/commit/76a23e9aa6bef27491fcd8a82a089eb4246a4ced))


### Dependencies

* **plugin-framework:** bump up blueprint lib to 0.46.0 ([598d66a](https://github.com/newstack-cloud/bluelink/commit/598d66a405aa5ab2451811af8925ddf3c59fa5cb))

## [0.8.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.7.0...plugin-framework/v0.8.0) (2026-05-07)


### Features

* **plugin-framework:** add config extraction helpers ([059f340](https://github.com/newstack-cloud/bluelink/commit/059f3400d4eaa63f87877c4d25b52ac6d0a5bca8))
* **plugin-framework:** add helpers to extract config maps and sequences ([757c568](https://github.com/newstack-cloud/bluelink/commit/757c5686f7cf38c10249bf0f6b2361391be63080))

## [0.7.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.6.0...plugin-framework/v0.7.0) (2026-05-05)


### Features

* **plugin-framework:** add utils and sub-framework for transformer plugins ([b7037ce](https://github.com/newstack-cloud/bluelink/commit/b7037ce10d8397101243e24878197625fc76896d))

## [0.6.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.5.2...plugin-framework/v0.6.0) (2026-04-28)


### Features

* **plugin-framework:** add transformer plugin utils and enhancements to transform interface ([79df70d](https://github.com/newstack-cloud/bluelink/commit/79df70df316bbe3d47152843fb81d8e9d29d926e))

## [0.5.2](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.5.1...plugin-framework/v0.5.2) (2026-04-25)


### Dependencies

* **plugin-framework:** bump up blueprint lib to 0.44.0 ([3e58d55](https://github.com/newstack-cloud/bluelink/commit/3e58d55b778eb46f187ba7d1aa87092697071984))

## [0.5.1](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.5.0...plugin-framework/v0.5.1) (2026-04-20)


### Dependencies

* **plugin-framework:** bump blueprint lib to 0.43.0 with removal policies ([2e7603f](https://github.com/newstack-cloud/bluelink/commit/2e7603f575c1799e1b6f8d4978fc461d9832b82c))

## [0.5.0](https://github.com/newstack-cloud/bluelink/compare/plugin-framework/v0.4.0...plugin-framework/v0.5.0) (2026-04-19)


### Features

* **plugin-framework:** add support for transformer abstract links and concrete link validation ([eef22e9](https://github.com/newstack-cloud/bluelink/commit/eef22e94154f3de58a70a37bd3116b35ae278ad2))


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
