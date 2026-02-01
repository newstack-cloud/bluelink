# Changelog

All notable changes to this project will be documented in this file.

## [0.41.0](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.40.0...blueprint/v0.41.0) (2026-02-01)


### Features

* **blueprint:** add deep mapping node validation in references ([25a45a5](https://github.com/newstack-cloud/bluelink/commit/25a45a59daddf5c8648e7bca3c0ca3b2cd8f8c9a))
* **blueprint:** add depends on list to schema tree ([8a73265](https://github.com/newstack-cloud/bluelink/commit/8a73265422b7e610b78899cfc62ed11491d95d23))
* **blueprint:** add support for accurately tracking field locations ([2e57b36](https://github.com/newstack-cloud/bluelink/commit/2e57b36dcbdf0bf2843a72c4e896fe3046df0c83))
* **blueprint:** add support for resolvable child blueprint export validation ([1a843b3](https://github.com/newstack-cloud/bluelink/commit/1a843b3b7eedffe12e6ce95be7d7033dc7b04d11))
* **blueprint:** add validation for link selector exclude list ([a16af10](https://github.com/newstack-cloud/bluelink/commit/a16af1003a9ad11654a5858072664cfdba18a73a))
* **blueprint:** add warnings for array index access in validation ([bf6e857](https://github.com/newstack-cloud/bluelink/commit/bf6e857920c0841540b80fd7aaaac028155bb0a3))


### Bug Fixes

* **blueprint:** add unmarshal yaml support for string lists ([591daa7](https://github.com/newstack-cloud/bluelink/commit/591daa744d3182de6a6c25bfe2ceca64bfbc69c4))
* **blueprint:** relax json string list parsing by skipping non-string values ([8e95b77](https://github.com/newstack-cloud/bluelink/commit/8e95b773b1bdbfb9805df172fdfc4f1e9e2a8579))


### Dependencies

* **blueprint:** bump google.golang.org/protobuf ([9b8eca6](https://github.com/newstack-cloud/bluelink/commit/9b8eca65a8983a3e2d83d92ffdd17d222169e9f4))

## [0.40.0](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.39.0...blueprint/v0.40.0) (2026-01-27)


### Features

* **blueprint:** add link selector exclude list and annotation value validation ([b9c366a](https://github.com/newstack-cloud/bluelink/commit/b9c366abee6ab22d5a6c0a2e0948188dffbcdd14))


### Dependencies

* **blueprint:** update common lib to version with similar string checks ([d4de9d4](https://github.com/newstack-cloud/bluelink/commit/d4de9d47534d0739c09cec7ad70cce979ef63b8b))

## [0.39.0](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.38.0...blueprint/v0.39.0) (2026-01-27)


### Features

* **blueprint:** add end position capture for mapping nodes and new registry methods ([36913e6](https://github.com/newstack-cloud/bluelink/commit/36913e6fe77d6cc2cf751d3636f256a8e18e78e5))
* **blueprint:** add field to more accurately map annotation to resource ([478e984](https://github.com/newstack-cloud/bluelink/commit/478e9841dad47d3a0d2d1b80dc3a1b25baa40142))
* **blueprint:** add improvements to diagnostics and errors ([a182c6c](https://github.com/newstack-cloud/bluelink/commit/a182c6c655871c287a228a7dae398fbc04b8ed92))
* **blueprint:** add support for single quotes in name accessors ([b179879](https://github.com/newstack-cloud/bluelink/commit/b179879ec75bf7919b5615f184acac9eee6f12ac))


### Bug Fixes

* **blueprint:** add improvements to errors and position tracking for errors ([77186c7](https://github.com/newstack-cloud/bluelink/commit/77186c7e6b38ec9d0fbc87480afaf6b151dbbab3))
* **blueprint:** add validation improvements and defensive checks ([ab0bea4](https://github.com/newstack-cloud/bluelink/commit/ab0bea421c19020e1a27a09506a99649b328aca3))

## [0.38.0](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.37.0...blueprint/v0.38.0) (2026-01-11)


### Features

* **blueprint:** add method to get batch of instances for list of ids or names ([2321597](https://github.com/newstack-cloud/bluelink/commit/2321597d8d048c4a753d2e190c8101ea7f8021a5))
* **blueprint:** add support for saving blueprint instances in batches ([ef2e18b](https://github.com/newstack-cloud/bluelink/commit/ef2e18b97c85481766773f68ef3a04ddbc0fe47b))

## [0.37.0](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.36.4...blueprint/v0.37.0) (2026-01-06)


### Features

* **blueprint:** add drift detection and reconciliation support for child blueprints ([347894f](https://github.com/newstack-cloud/bluelink/commit/347894fea55e338d924eef57886a4adf0bfda97e))
* **blueprint:** add helpers to support rollbacks for new and existing deployments ([9ff363b](https://github.com/newstack-cloud/bluelink/commit/9ff363b3541d4af7fd902f99d6cfa4eda0dbbc41))
* **blueprint:** add method to list instances to state container ([239b022](https://github.com/newstack-cloud/bluelink/commit/239b022a61118f252a256e65b8af4ae4237b70d8))
* **blueprint:** add safe to rollback check for links ([f486673](https://github.com/newstack-cloud/bluelink/commit/f4866737a85f7ef5d4d1d9a400f21ae049f011de))
* **blueprint:** add safe to rollback checks for resources ([881fe79](https://github.com/newstack-cloud/bluelink/commit/881fe79068ae666408208265f43bf64ab32ec2dd))
* **blueprint:** add support for advanced reconciliation and drift checks ([b889111](https://github.com/newstack-cloud/bluelink/commit/b8891116050e5f2c5d11d48e37853cb65e6f28be))
* **blueprint:** add support for sorting resource schema arrays for comparison ([d8e8ed6](https://github.com/newstack-cloud/bluelink/commit/d8e8ed6b601c02627f7d4d984e9c6be953469d87))
* **blueprint:** add support for system-level tagging ([74122ad](https://github.com/newstack-cloud/bluelink/commit/74122ad9cf577eb6775494829b768272617a1c74))
* **blueprint:** integrate full support for rollbacks  and pre-rollback state capture ([e089295](https://github.com/newstack-cloud/bluelink/commit/e089295bcedb6001b8ff1434d08465b2307e25af))


### Bug Fixes

* **blueprint:** add fixes to change set output and add computed fields to resource state ([06b2b08](https://github.com/newstack-cloud/bluelink/commit/06b2b08dad9e509744afa22c028282e891960895))
* **blueprint:** add missing json struct tags to reconciliation types ([053f155](https://github.com/newstack-cloud/bluelink/commit/053f155d0067bfa22732aff27afd0cd9ab19e67d))
* **blueprint:** add missing tagging config to drift checker ([4d59e61](https://github.com/newstack-cloud/bluelink/commit/4d59e61cac273db005a872cd33efc9c93a6d250c))
* **blueprint:** add resource name to get external state input ([56d8434](https://github.com/newstack-cloud/bluelink/commit/56d8434ec331d166f08b01e6c21af81eb9405c5f))
* **blueprint:** capture child blueprint final status and correct messaging for interruptions ([54dfdc1](https://github.com/newstack-cloud/bluelink/commit/54dfdc196fdd70e48e5f3c4149f6abd02671d249))
* **blueprint:** ensure resource spec and child attachments are persisted even for failed deployments ([c435296](https://github.com/newstack-cloud/bluelink/commit/c435296c9801fa9fb3ca2db6a4d84be72a3172ae))
* **blueprint:** improve failure handling for deployment orchestration ([3bedabc](https://github.com/newstack-cloud/bluelink/commit/3bedabcc1107edb05f1a71b3b4e2e9a1f974bd98))
* **blueprint:** improve failure handling for deployment orchestration process ([d88bd48](https://github.com/newstack-cloud/bluelink/commit/d88bd48362247e6a46a512e7c1d7bb77d9b49a3d))

## [0.36.4](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.36.3...blueprint/v0.36.4) (2025-12-16)


### Features

* **blueprint:** add behaviour to reverse a change set ([bbbdb1a](https://github.com/newstack-cloud/bluelink/commit/bbbdb1acd7983580eace24ee397d9456bf60dfb5))
* **blueprint:** add improvements to blueprint container ([39c5e2e](https://github.com/newstack-cloud/bluelink/commit/39c5e2e3a1e01c603cb521e7a08bf4bbf7b56cae))


### Bug Fixes

* **blueprint:** ensure empty mapping nodes are treated correctly for protobuf conversion ([1822d2f](https://github.com/newstack-cloud/bluelink/commit/1822d2fe03cede073bcf63d6c5d60fc2009bdc9a))

## [0.36.3](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.36.2...blueprint/v0.36.3) (2025-12-13)


### Bug Fixes

* **blueprint:** initialise missing maps in initial instance state ([8bb3766](https://github.com/newstack-cloud/bluelink/commit/8bb3766e3916e5dffbcbbacde0935c0c6182503f))

## [0.36.2](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.36.1...blueprint/v0.36.2) (2025-12-10)


### Features

* **blueprint:** add support for resolving relative child blueprints from local fs ([e3806b3](https://github.com/newstack-cloud/bluelink/commit/e3806b36359d2f0dc84a10f6aed725830f08987c))


### Bug Fixes

* **blueprint:** add fix for nil pointer derefs for change staging ([f4a7932](https://github.com/newstack-cloud/bluelink/commit/f4a7932c4b2a1a80e0cd5afdfaf1a44b1a8a1f67))
* **blueprint:** add fix for validating optional link annotations ([7983c7f](https://github.com/newstack-cloud/bluelink/commit/7983c7fbcad9535deba4d80ca793bac6a2af3075))

## [0.36.1](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.36.0...blueprint/v0.36.1) (2025-12-09)


### Bug Fixes

* **blueprint:** ensure link context is set for method calls to link implementations ([b3c18df](https://github.com/newstack-cloud/bluelink/commit/b3c18df25b2e2986bd81fd76b424ae44c3ce399d))


### Dependencies

* **blueprint:** bump the go-deps group ([ef381e8](https://github.com/newstack-cloud/bluelink/commit/ef381e8d881fea096bc32e3267f984a1236ecee0))

## [0.36.0] - 2025-11-03

### Added

- Adds support for the `coalesce` function to the core provider. This function allows for the selection of the first non-none value from a list of values.
- Adds support for `none` values in the substitution language as per the latest updates to the blueprint specification.

### Fixed

- Corrects the substitution engine behaviour for passing variadic arguments to functions. Variadic arguments are now passed as a single slice of values instead of individual arguments so that function implementations can distinguish between variadic and non-variadic arguments. When there is a known number of expected arguments, they are passed as individual arguments that can be extracted with the call context helpers, this doesn't work for variadic arguments as the number of arguments is not known until the function is called. Wrapping variadic arguments in a single slice means that functions do not need to know the number of arguments they are expected to receive in advance.

## [0.35.0] - 2025-10-18

### Added

- Adds support for the `first` function to the core provider. This function allows for the retrieval of the first non-empty value from a list of values.
- Adds support for the `lookup` function to the core provider. This function allows for the retrieval of a value from a mapping by key, falling back to a default value that must be provided.
- Adds support for the `cidrsubnet` function to the core provider. This function allows for the calculation of a subnet from a CIDR block and a prefix length.
- Adds support for the `sha256` function to the core provider. This function allows for the calculation of the SHA-256 hash of a string or byte array.
- Adds support for the `md5` function to the core provider. This function allows for the calculation of the MD5 hash of a string or byte array.
- Adds support for the `sha1` function to the core provider. This function allows for the calculation of the SHA-1 hash of a string or byte array.
- Adds support for the `uuid` function to the core provider. This function allows for the generation of a UUID.
- Adds support for the `base64encode` function to the core provider. This function allows for the encoding of a string or byte array to a base64 string.
- Adds support for the `base64decode` function to the core provider. This function allows for the decoding of a base64 string to a byte array.
- Adds support for the `min` function to the core provider. This function allows for the finding of the minimum value in a list of values.
- Adds support for the `max` function to the core provider. This function allows for the finding of the maximum value in a list of values.
- Adds support for the `abs` function to the core provider. This function allows for the finding of the absolute value of a number.
- Adds support for the `file` function to the core provider. This function allows for the retrieval of the contents of a file from the file system and remote locations. Remote locations are supported using a `{scheme}://` prefix in the path parameter and the host applications are responsible for implementing and documenting the supported remote locations and their schemes.
- Adds support for the `utf8` function to the core provider. This function allows for the conversion of a string to a UTF-8 encoded byte array.
- Adds support for the `http_resource` function to the core provider. This function allows for the retrieval of the contents of a resource from a HTTP endpoint.
- Adds support for the `if` function to the core provider. This function allows for conditional logic for choosing between two values based on a condition.
- Adds support for storing byte arrays at runtime to allow for substitution functions that either take binary data as input or return binary data as output. All binary outputs will be encoded as UTF-8 strings automatically when used as the value of a blueprint element field. Serialisation of the `core.ScalarValue` struct will always convert byte arrays to UTF-8 strings, once serialised, the distinction between bytes and strings is lost and the value will be deserialised as a `core.ScalarValue` with a `StringValue` field.

## [0.34.1] - 2025-09-28

### Fixed

- Removed list error action types as the early versions of clients such as the bluelink CLI will not support listing available resource types, links, data source types and custom variable types.

## [0.34.0] - 2025-09-21

### Added

- Adds deeper integration of error context in key load and run errors, focused around actions that the user can take to resolve issues with resources, data sources and variables.

## [0.33.0] - 2025-09-16

### Added

- Adds support for error context in load, run errors and diagnostics to allow for more user-friendly error messages with suggestions for actions that the user can take to resolve the error.

## [0.32.0] - 2025-07-26

### Added

- Adds a new `TrackDrift` field to the `provider.ResourceDefinitionsSchema` struct to allow provider developers to mark computed resource fields that should be tracked for drift. This is useful for detecting changes to derived values from a resource deployment that should be considered drift. This change also includes updates to the resource change generator and drift checker to ensure that computed fields that are marked as `TrackDrift` are included in resource change sets and drift checks.

## [0.31.1] - 2025-07-21

### Fixed

- Adds correction to make sure that the current link state is passed into the `UpdateResourceA`, `UpdateResourceB` and `UpdateIntermediaryResources` methods of the `provider.Link` implementations. This is to ensure that the link implementation has access to the current state of the link when applying updates to activate, update or deactivate the link.

## [0.31.0] - 2025-07-20

### Added

- Adds the `core.MappingNodeFields` and `core.MappingNodeItems` helper functions to provide a more convenient and concise way to build mapping nodes that represent maps/objects and arrays.

## [0.30.1] - 2025-07-19

### Fixed

- Add missing `ProviderContext` field to the `AcquireResourceLockInput` struct used to acquire resource locks.

## [0.30.0] - 2025-07-19

### Added

- **Breaking change** -  Adds support for resource locking to prevent links applying updates to the same resource at the same time that would lead to conflicts and data races. Links are not orchestrated like resources are with a dependency graph, links are deployed asynchronously as soon as the two primary resources linked together have been deployed, this means there are no guarantees that multiple links operating on the same resources will not try to update the same resources at the same time.

  This is primarily an issue for links that update existing resources in the same blueprint that are considered intermediary resources for a link. This includes breaking changes to the resource registry and the interfaces of services passed in as a part of the input for the `UpdateIntermediaryResources` method. Link implementations are now able to acquire locks scoped to the blueprint instance for a specific resource when applying updates to ensure that two links are not updating the same resource at the same time.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.29.0] - 2025-07-19

### Added

- **Breaking change** - Changes the name of the `core.InjectPathValueReplace` function to `core.InjectPathValueReplaceFields` to better reflect its purpose of replacing fields in a mapping node as the `core.InjectPathValue` function replaces elements in an array but not fields in a map/object mapping node.
- Adds support for path array selectors for mapping paths used for searching for values in mapping nodes along with injecting values into mapping nodes. The `[@.<key> = "<value>"]` syntax can be used to select an item in an array that has a specific key matching a provided value. This is useful for working with arrays of objects in mapping nodes when patching updates to specific objects in an array by a unique identifier. This is the case for things like cloud provider IAM policies where the representation for policy documents is an array of objects with a unique identifier instead of a mapping of IDs to policy documents. This behaviour is supported for the `core.GetPathValue`, `core.InjectPathValue`, `core.InjectPathValueReplaceFields` and `core.PathMatchesPattern` functions.

_Breaking changes will occur in early 0.x releases of this framework._

### Fixed

- Corrects the mapping node injection behaviour to make sure entire structures are not replaced with an empty array or mapping when injecting in "replace fields" mode. Arrays and mappings should only be created if the value in the injection path does not already exist in the structure that the value is being injected into.

## [0.28.0] - 2025-07-17

### Added

- Adds support for the `IgnoreDrift` field in resource schema definitions. This allows plugin developers to mark fields in resource schemas that should not be checked for drift.
- Adds `DestroyBeforeCreate` field to resource spec definitions. This is to indicate in documentation and tools built on top of the blueprint framework that a resource should be destroyed before it is created when a resource must be recreated as a part of a deployment.

## [0.27.0] - 2025-07-13

### Added

- Adds support for marking fields as sensitive in resource and data source schema definitions. This also includes updates to the generate changes functionality to ensure that field changes are marked as sensitive when the field in the source schema is marked as sensitive.

## [0.26.0] - 2025-07-03

### Added

- Adds a new field to link state that holds a mapping of fields in any resource in the same blueprint to the field path in the link data. This is primarily useful in that it allows the drift checker to overlay link data in the same blueprint on resource state so that updates made by links to any resource in the same blueprint are not picked up as drift. This is essential for the concept of links as side effects that can update resources linked together along with existing resources in the same blueprint that act as intermediaries, without this behaviour, any changes that links make will be consistently picked up as drift and if you relax the drift checker then it will be very difficult to track the current state and the abstraction of links would become an unwieldy addition that would make things less predictable. With the combination of the link data projection and link changes included when staging changes, there is a clear picture of what the current state should be looking at the resource spec data as defined by the user and the updates made by linking resources together. Drift detection now takes link changes into account, based on a similar picture to what the user would see when staging changes.

## [0.25.0] - 2025-07-02

### Added

- **Breaking change** - Adds the `LookupResourceInState` abd `HasResourceInState` methods to the `resourcehelpers.Registry` interface. This allows for looking up resources in the state container by their external ID. This is useful for link plugin implementations that use existing resources as intermediaries that get updated as a part of the link implementation.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.24.1] - 2025-06-24

### Fixed

- Corrects the go module path in the `go.mod` file to `github.com/newstack-cloud/bluelink/libs/blueprint` for all future releases.

## [0.24.0] - 2025-06-22

### Added

- Adds support for complex static values or user-defined objects and arrays that can contain dynamic elements. As per the latest iteration of the blueprint specification the way value content is defined has been expanded to support complex mappings and arrays that can either be static or can contain substitutions for dynamic elements. This allows for use cases such as defining reusable policy documents for a cloud provider without plugin developers needing to create virtual resource types to make for a better experience in reusing policies and take on the maintenance burden of the complexities involved in virtual resources. Values simplify the plugin developer experience, not having to worry about virtual resources to organise unwieldy resource specs into smaller components and allows the practitioners defining blueprints with the ability to break down larger resource specs using values.

## [0.23.0] - 2025-06-21

### Added

- Adds changes to ensure that the user-defined blueprint instance name is passed to actions used in the deployment lifecycle and for drift checking. This is especially useful for logging and debugging purposes within plugins along with providing plugins with the ability to create meaningful unique names for certain kinds of resources (e.g. AWS IAM roles) that the user is not required to provide a name for in the blueprint document.

## [0.22.0] - 2025-06-18

### Added

- **Breaking change** - Adds support for more advanced filter field definitions including supported operators, conflicts and descriptions.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.21.0] - 2025-06-17

### Added

- Adds support for exporting all fields from a data source with `exports: "*"` as per the latest updates to the blueprint specification. This work includes behaviour to enhance validation to look up the data source exports specification when exporting all fields along with modifications to the subengine resolver to populate the resolved exports map from the data source spec field schema.

## [0.20.0] - 2025-06-17

### Added

- Adds support for describing filter fields separately from data source schema fields to account for documenting fields that can be used for filtering that are not a part of the data source schema.

## [0.19.0] - 2025-06-17

### Added

- Adds support for multiple data source filters as per the latest updates to the blueprint specification. The specification now allows for multiple filters that will be combined together using a logical AND operation to filter resources in a data source.

## [0.18.0] - 2025-06-11

### Added

- Adds the new `core.DiagnosticRangeFromSourceMeta` helper function to create a diagnostic range from metadata about the location of an element in a source blueprint document. This is useful for implementations of the `provider.Provider` or `transform.SpecTransformer` interfaces that need to create diagnostics for custom validation.

## [0.17.1] - 2025-06-10

### Fixed

- Corrects the go module path in the `go.mod` file to `github.com/newstack-cloud/bluelink/libs/blueprint` for all future releases.

## [0.17.0] - 2025-06-10

### Added

- Adds helpers to the `core` package for extracting slices and maps from `MappingNode`s. This includes the `*SliceValue` and `*MapValue` functions where `*` represents a scalar type that can be one of `String`, `Int`, `Float` or `Bool`. If an empty mapping node or one that does not represent a slice or map is passed to these functions, they will return an empty slice or map of the appropriate type. When values of other types are encountered in a map or slice, empty values of the target type will be returned.

## [0.16.0] - 2025-06-08

### Fixed

- Adds missing behaviour to escape regular expression special characters when forming dynamic config field name patterns.

### Added

- Adds convenience methods to the `core.PluginConfig` map wrapper to extract slices and maps from config
  value prefixes. The `SliceFromPrefix` and `MapFromPrefix` methods allow for the extraction of slices and maps from config key prefixes that represent a map or slice of scalar values. For more complex structures, the `GetAllWithSlicePrefix` and `GetAllWithMapPrefix` methods provided will filter down the config map to only include keys that start with an array or map prefix along with extra metadata such as ordering of keys for a slice representation.

## [0.15.0] - 2025-06-05

### Fixed

- Adds fix to ensure that errors are separated from warnings and info diagnostics in the result of the `ValidateLinkAnnotations` function so that the error is not ignored by the validation process. The validation process separates error froms other kinds of diagnostics so that it is easier to evaluate if validation has failed overall when loading a blueprint.

### Added

- Adds support for custom validation functions that can be defined for individual resource definition schema elements. This allows custom value-based validation and conditional validation based on other values in the resource as defined in the source blueprint. This validation is limited to scalar types (integers, floats, strings and booleans) and will not be called for strings that contain `${..}` substitutions.

## [0.14.0] - 2025-06-04

### Added

- Adds support for custom validation functions in link annotation definitions for provider plugins. The validation function takes a key and a value to allow for advanced standalone validation (such as a regexp pattern or string length constraints).

## [0.13.0] - 2025-06-04

### Added

- Adds support for custom validation functions in config field definitions for provider and transformer plugins. This validation function takes a key, value and a reference all the plugin config to allow for advanced standalone validation (such as a regexp pattern) as well as validation for things like conditionally required fields that depend on other values in the plugin config. This also includes a new helper type alias for a config map that allows for retrieving all config values that have a certain prefix which will be very useful for namespaced config variables that emulate more complex structures where conditional validation will often depend on other config values under a specific namespace.

## [0.12.0] - 2025-06-04

### Fixed

- Ensure annotations are resolved as scalar types other than strings. As per the blueprint specification, annotation values can be strings, integers, floats or booleans. The implementation before this commit would resolve all exact values as strings, other types would only be resolbed if a `${..}` substitution is used for an annotation value. The changes included with this commit will resolve literal values defined for annotations in a blueprint to the more precise scalar type.
- Adds missing value type checks to plugin config field validation.

### Added

- Adds functionality to validate annotations used for links. This validation kicks in after a chain/graph of link nodes has been formed and has already been checked for cycles to ensure check annotations used in resources against the schema for annotations provided by the provider plugin link implementations that enable the links between resources.

## [0.11.0] - 2025-05-31

### Added

- Adds a new set of value constraints to the `provider.ResourceDefinitionsSchema` struct to allow providers and transformers to define specific constraints for values in resource specs used in both concrete and abstract resource types. This commit includes updates to validation to carry out strict checks when exact values are provided and produce warnings for string interpolation where the value is not known until the substitution is resolved during change staging or deployment.
  - Adds `Minimum` and `Maximum` fields for numeric types.
  - Adds `Pattern` field for strings to match against a Go-compatible regular expression.
  - Adds `MinLength` and `MaxLength` fields for strings (character count), arrays and maps.

## [0.10.0] - 2025-05-30

### Added

- Adds an `AllowedValues` field to the `provider.ResourceDefinitionsSchema` struct to allow providers and transformers to define enum-like constraints for values in resource specs used in both concrete and abstract resource types. This commit includes updates to validation to carry out strict checks when scalar values are provided and produce warnings for string interpolation where the value is not known until the substitution is resolved during change staging or deployment.

## [0.9.0] - 2025-05-16

### Changed

- **Breaking change** - Updates the Celerity blueprint document version for validation to `2025-05-12`.
- **Breaking change** - Updates the Celerity transform version to `2025-08-01` as the final initial version string for the Celerity application transform in anticipation of a release of Celerity as a whole in late summer/early autumn 2025.

### Added

- **Breaking change** - Adds full support for JSON with Commas and Comments. The latest update to the Blueprint specification switches out plain JSON for JSON with Commas and Comments. This allows for a more human-readable format that is easier to work with for the purpose of configuration. This release adds full support for this format along with changes to the default JSON parse mode for the schema loading functionality to track line and column numbers using the coreos fork of the `encoding/json` package.
- Adds new `AllowAdditionalFields` property to the `core.ConfigDefinition` struct used for plugin config variables.
- Adds functionality to populate defaults and validate plugin config.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.8.0] - 2025-05-02

### Added

- Adds a stub resource in the core provider to allow loading of blueprints that have no real resources. This is used as a work around for the design of the blueprint loader and state container to be able to destroy blueprint instances without requiring the user to provide a source blueprint document. This is because the destroy functionality does not use or need any of the data from a loaded blueprint as it operates with a provided change set and the current state of a blueprint instance.

## [0.7.2] - 2025-05-02

### Fixed

- Adds missing JSON tag to the `source.Meta.EndPosition` field to ensure field names are "lowerCamelCase" when serialised to JSON.

## [0.7.1] - 2025-05-02

### Fixed

- Adds missing JSON tags to serialise `source.Position` fields with lower case field names.

## [0.7.0] - 2025-04-17

### Added

- Adds JSON tags to the `provider.RetryPolicy` struct to ensure that the retry policy is correctly serialised and deserialised when using JSON encoding.

## [0.6.0] - 2025-04-16

### Added

- **Breaking change** - Adds a new `LookupIDByName` method to the `state.Container` interface to allow for looking up the ID of a blueprint instance by its name.
- **Breaking change** - Updates the `Deploy`, `Destroy` and `StageChanges` methods of the blueprint container implementation to accept an `InstanceName` field that allows users to provide a name instead of (or in addition to) an ID. This makes for a better user experience when working with blueprints where they can use a name instead of an ID to identify the blueprint instances they are working with.

## [0.5.0] - 2025-04-14

### Added

- Adds `ListLinkTypes` method to provider interface to allow tools and applications (e.g. Deploy Engine) that use providers to list the link types implemented by the provider.

## [0.4.1] - 2025-04-14

### Fixed

- Adds missing required field to the `function.ValueTypeDefinitionObject` type that defines an object type used as a function parameter or return type.

## [0.4.0] - 2025-04-14

### Added

- **Breaking change** - Add method to export all provider config variables from link contexts.
- Adds helper functions to create `core.ScalarValue` structs from Go primitive types.
- Adds zap logger adaptor for the `core.Logger` interface. This allows for the use of zap logger implementations with the blueprint framework core logger interface.
- **Breaking change** - Adds example method for resources, data sources and custom variable types.
- Adds functionality to allows link implementations to call back into the blueprint framework host to deploy resources registered with the host.
- Adds stabilisation check method to resource registry to allow the resource registry to fulfil a new `ResourceDeployService` interface introduced to allow link plugins to call back into the host to deploy known resources for intermediary resources managed by a link implementation.
- Adds a convenience error interface that allows the extraction of a list of failure reasons without having to check each possible provider error type.
- Adds a new provider.BadInput error type that can be used to help distinguish between input and unexpected errors when providing feedback to the user. _Bad input errors are not handled in any special way in the blueprint container that manages deployments, it must be wrapped in specialised errors to be handled correctly for deployment actions._
- Adds new public util functions that provide a consistent entry point for generating logical link names and link type identifiers.

### Updated

- Adds a new `Secret` field to the plugin config definitions struct. This allows plugin developers to indicate that a config variable is a secret and should be treated as such by the blueprint framework. This is useful for sensitive information such as passwords or API keys.
- Make plugin function call stack thread-safe to be able to handle calls from multiple goroutines.
- Enhances plugin interfaces with methods to provide rich documentation.
- Simplifies resource registry interface when it comes to waiting for resource stability. This is possible through the introduction of a convenience layer to the resource registry interface and core implementation to allow callers to wait for a resource to stabilise using a single boolean option instead of having to set up a polling loop and call `HasStabilised` on an interval to do so.
- **Breaking change** - Adds update to make creating a new plugin function call context a part of the public interface.
- **Breaking change** - Alters the public interface of existing provider errors to make the error check functions more idiomatic where you write to a target error and check the error type in the same action.
- Updates some previously private protobuf conversion methods to be public so that gRPC services (such as the deploy engine plugin system) can reuse existing conversion behaviour.
- Enhances provider-specific error types with child errors so they can hold more structured information.

### Removed

- Removes the `SetCurrentLocation` from the function call context as it is never used, each context gets its own call context with a current location that does not need to change for the lifetime of the function being called.

### Fixed

- Corrects the name of the transformer context field in the `transform.AbstractResourceGetExamplesInput` struct.
- Corrects behaviour to convert from protobuf to blueprint framework types so that a mapping node type with all fields set to nil is treated correctly when the mapping node being converted is optional.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.3.3] - 2025-02-27

### Fixed

- Adds missing `CurrentResourceSpec` and `CurrentResourceMetadata` fields to the `ResourceGetExternalStateInput` struct that is passed into the `GetExternalState` method of the `provider.Resource` interface. This allows the provider to access the current resource spec and metadata when determining how to locate and present the "live" state of the resource in the external system.

## [0.3.2] - 2025-02-26

### Updated

- **Breaking change** - Renamed the `StabilisedDependencies` method of the `provider.Resource` interface to `GetStabilisedDependencies` to be consistent with the names of other methods to retrieve information about a resource.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.3.1] - 2025-02-25

### Fixed

- Corrects error code used in the helper function to create export not found errors. This also adds a missing switch case for the display text to show for the export not found error code.

## [0.3.0] - 2025-02-25

### Added

- Helper functions to create and check for export not found errors to be used as a part of the state container interface. Implementations of the state container interface should use these functions to ensure that consistent error types are returned when a requested export is not present in a given blueprint instance.

## [0.2.3] - 2025-02-14

### Updated

- **Breaking change** - Renamed the `Remove` method of the children state container to `Detach` to be consistent with the `Attach` method and to highlight that the method does not completely remove the child blueprint state but removes the connection between the parent and child blueprints.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.2.2] - 2025-02-09

### Fixed

- Adds a workaround to ensure Go does not try to zip contents of test data and snapshot directories when importing the package into a project. This workaround includes adding an empty `go.mod` file to every directory that should be ignored. Without this fix, the package could not be imported into packages due to unusual characters in the snapshot file names generated by the cupaloy package used for snapshot testing. It is also good practise to ignore these directories as they are not required for projects to make use of the package.

## [0.2.1] - 2025-02-04

### Updated

- **Breaking change** - Simplifies redundant entity type prefixed field names in state structures. (e.g. `ResourceName` -> `Name`) The exception to this change involves the id fields due to the `ID` name being used for the method that fulfils the Element interface. For this reason ResourceID, InstanceID and LinkID will remain in the Go structs but will be serialised to "id" when marshalling to JSON.'

_Breaking changes will occur in early 0.x releases of this framework._

## [0.2.0] - 2025-01-31

### Updated

- **Breaking change** - Removes redundant instance ID arguments from state container methods when interacting with resources and links by globally unique identifiers.
- Updates the in-memory state container implementation used in automated tests to store resource and link data in a way that it can be directly accessed when given a globally unique identifier.

_Breaking changes will occur in early 0.x releases of this framework._

## [0.1.0] - 2025-01-29

### Added

- Functionality to load, parse and validate blueprints that adheres to the [Blueprint Specification](https://www.celerityframework.io/docs/blueprint/specification).
- Functionality to stage changes for updates and new deployments of a blueprint.
- Functionality to resolve substitutions in all elements of a blueprint.
- Functionality to deploy and destroy blueprint instances based on a set of changes produced during change staging.
- An interface and data types for persisting blueprint state.
- Interfaces for interacting with resource providers that applications can build plugin systems on top of.
- A set of core functions that can be used with substitutions along with tools for creating custom functions through a provider plugin.
- Functionality to check whether the "live" external state of a resource or set of resources in a blueprint matches the current state persisted with the blueprint framework.
