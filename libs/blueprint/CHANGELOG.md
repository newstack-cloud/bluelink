# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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
