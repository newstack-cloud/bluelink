# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.36.1](https://github.com/newstack-cloud/bluelink/compare/blueprint/v0.36.0...blueprint/v0.36.1) (2025-12-02)


### Features

* **blueprint:** add "first" substitution function ([784d501](https://github.com/newstack-cloud/bluelink/commit/784d501ab3d701e5865699d7e619e5bde7629279))
* **blueprint:** add "lookup" substitution function ([518a6f9](https://github.com/newstack-cloud/bluelink/commit/518a6f9bc942e6f1dd4429aba92a6b039cfacfa7))
* **blueprint:** add a new set of value constraints to resource spec schemas ([93dc344](https://github.com/newstack-cloud/bluelink/commit/93dc3449c3e25bd9ace67e435e87f300e1b7970f))
* **blueprint:** add abs function ([a4b631c](https://github.com/newstack-cloud/bluelink/commit/a4b631c1178507bbbb5285387d4e580bc0f41396))
* **blueprint:** add allowed values to resource definitions schema ([1f1b973](https://github.com/newstack-cloud/bluelink/commit/1f1b9735fa6a55ea43f3736e576482ed86297886))
* **blueprint:** add b64 encode and decode functions ([e0e74b5](https://github.com/newstack-cloud/bluelink/commit/e0e74b5e03f00e2b263df3be2fbb4af3e444492e))
* **blueprint:** add behaviour to export all transformer and context variables ([4bcefa6](https://github.com/newstack-cloud/bluelink/commit/4bcefa6838fe4dab658cdfdba51f36bc598b1d23))
* **blueprint:** add change staging for blueprint-wide metadata ([3722400](https://github.com/newstack-cloud/bluelink/commit/37224007beb48523df93a44652f577bd99579dd4))
* **blueprint:** add complete deployment functionality ([63071c5](https://github.com/newstack-cloud/bluelink/commit/63071c501ae64e8a9662ce362991b08a49294757))
* **blueprint:** add convenience error interface for extracting failure reasons ([b27a5ec](https://github.com/newstack-cloud/bluelink/commit/b27a5ec68ca6c7b6fc26cad363ec80f40853b104))
* **blueprint:** add core helpers for extracting maps and slices from mapping nodes ([e1a9ed0](https://github.com/newstack-cloud/bluelink/commit/e1a9ed0bde37f741b4d94fa02631fd80964962db))
* **blueprint:** add custom validation for individual resource definition schema elements ([a44e36d](https://github.com/newstack-cloud/bluelink/commit/a44e36d16898a6f960f96942e85489c7b36bb5bf))
* **blueprint:** add descriptions field for filter fields ([5ec53a3](https://github.com/newstack-cloud/bluelink/commit/5ec53a326cc0ee1ae305668ddba96927316abee9))
* **blueprint:** add drift check functionality for resources ([1061468](https://github.com/newstack-cloud/bluelink/commit/1061468db418f7b485eb1dac6d10e94fb883eb2a))
* **blueprint:** add examples method for resources ([8bb6538](https://github.com/newstack-cloud/bluelink/commit/8bb6538f5a9d7c17f9db8bf4ce40f0454dcf2999))
* **blueprint:** add export not found error ([95f56ce](https://github.com/newstack-cloud/bluelink/commit/95f56ce54aa74c437af1781f159db90a1a584668))
* **blueprint:** add extensible file function implementation ([9cf22d5](https://github.com/newstack-cloud/bluelink/commit/9cf22d57ff26420350c3a6f9c2828f1fd21991b7))
* **blueprint:** add final modifications to enable export all fields behaviour ([03fb8c1](https://github.com/newstack-cloud/bluelink/commit/03fb8c112cdb7ec2e1bfd35460720e805cbf2e49))
* **blueprint:** add full support for json with commas and comments ([08ea4be](https://github.com/newstack-cloud/bluelink/commit/08ea4be5879fc3872698bb91736fd20437284ab1))
* **blueprint:** add functionality to allow link plugins to deploy known resources ([048f16f](https://github.com/newstack-cloud/bluelink/commit/048f16f7465811c23e91d28852a3a6b6702982f5))
* **blueprint:** add functionality to look up resources by external id in resource registry ([5548e3e](https://github.com/newstack-cloud/bluelink/commit/5548e3e25ee1d81366ee3bf3afd85e0e2e713ceb))
* **blueprint:** add functionality to populate defaults and validate plugin config ([9f2b9a2](https://github.com/newstack-cloud/bluelink/commit/9f2b9a2a893af91eb693b59fd1c4541d4b5c775d))
* **blueprint:** add helper functions to create scalar structs from go values ([40cbf0a](https://github.com/newstack-cloud/bluelink/commit/40cbf0a9e2f09e30a68259a7f57d253f07360f07))
* **blueprint:** add helpers for creating mapping node maps and arrays ([1b28cc1](https://github.com/newstack-cloud/bluelink/commit/1b28cc1932eb0cd6fad8125dbed0cd60e27200da))
* **blueprint:** add implementation of child blueprint deployer ([a6d2bf3](https://github.com/newstack-cloud/bluelink/commit/a6d2bf3f367b71a3e2ec45f084c1f2b2d7a8a430))
* **blueprint:** add json tags to retry policy to allow for serialisation ([b206919](https://github.com/newstack-cloud/bluelink/commit/b206919912b05ed82a2d943438fb3c2b28f111e4))
* **blueprint:** add link to resource data mappings in a blueprint instance ([8fc505e](https://github.com/newstack-cloud/bluelink/commit/8fc505e7da5ce38bf98a0f0ff4e92cd4e7bb7346))
* **blueprint:** add logger interface and integrate logging ([bf32564](https://github.com/newstack-cloud/bluelink/commit/bf32564378d4ce65aded5b82320b44ae550f810e))
* **blueprint:** add max and min functions ([30d3aea](https://github.com/newstack-cloud/bluelink/commit/30d3aeac988c7616efc9adfd1e2fa4ad0a96ec26))
* **blueprint:** add md5 function ([6912833](https://github.com/newstack-cloud/bluelink/commit/6912833c75bdcec4b09ea9ebc061ea0414bf41fc))
* **blueprint:** add method to export all provider config variables from link context ([a58599b](https://github.com/newstack-cloud/bluelink/commit/a58599bca6908dbc5e7924a7085d0f97e909b27c))
* **blueprint:** add method to get examples for custom var type plugin ([5f61284](https://github.com/newstack-cloud/bluelink/commit/5f61284152a055aa59b513158140e0f55a6033da))
* **blueprint:** add methods to plugin config to extract complex structures ([567415a](https://github.com/newstack-cloud/bluelink/commit/567415a45ad1e818df91f426cd1fdafb822e9e6e))
* **blueprint:** add new allow additional fields property to config definition ([0eaffb3](https://github.com/newstack-cloud/bluelink/commit/0eaffb399b01f684d613ea8576b2638a74e45dd5))
* **blueprint:** add new fields to function plugin types for richer docs ([fa01240](https://github.com/newstack-cloud/bluelink/commit/fa01240a2fe6f3505cb3dbdb99687ab06a1fcb4d))
* **blueprint:** add new methods to abstract resources for richer docs ([bc1d1dd](https://github.com/newstack-cloud/bluelink/commit/bc1d1dd62d942dfe38b82932b1ddb40c79597648))
* **blueprint:** add new utils, enhance errors and clean up unused behaviour ([0f328aa](https://github.com/newstack-cloud/bluelink/commit/0f328aa45993c791d5eda877a9a93db4b138df18))
* **blueprint:** add provider method to list link types ([3bdf377](https://github.com/newstack-cloud/bluelink/commit/3bdf3770dd861ab967a74880c5ff9fc9c7030450))
* **blueprint:** add public protobuf conversion methods and improve provider errors ([ecb72cb](https://github.com/newstack-cloud/bluelink/commit/ecb72cba2d8905febfdbdff601cce32ea7877208))
* **blueprint:** add resource stabilisation polling behaviour ([5d2c980](https://github.com/newstack-cloud/bluelink/commit/5d2c98048661cd7f435bfb2862dec89ca2cc7b6c))
* **blueprint:** add retry behaviour for resource stabilisation checks ([3d561a5](https://github.com/newstack-cloud/bluelink/commit/3d561a5b286279a716d845bc0495b9f6ec185f3e))
* **blueprint:** add retry support when fetching from data sources ([f76bbea](https://github.com/newstack-cloud/bluelink/commit/f76bbea27850d1ecd39c86f8dff3231aceba7c3f))
* **blueprint:** add rich, structured error context for errors and diagnostics ([ad9ff69](https://github.com/newstack-cloud/bluelink/commit/ad9ff6912ce1cd61590cef31b86c1d65fb19b61d))
* **blueprint:** add secret to plugin config definitions ([dd9733e](https://github.com/newstack-cloud/bluelink/commit/dd9733eb06ad7224bd5ec5a7f199becb183f9424))
* **blueprint:** add sensitive fields to resource and data source schemas ([7f88ccb](https://github.com/newstack-cloud/bluelink/commit/7f88ccbf8522de87bdd2379fb8ac3213d96df68b))
* **blueprint:** add sha1 function ([913aa58](https://github.com/newstack-cloud/bluelink/commit/913aa58fd921e07bfdb2d1232db0029a90e96860))
* **blueprint:** add sha256 function ([2227ed9](https://github.com/newstack-cloud/bluelink/commit/2227ed93291fdf697cab03e33de1d0039cd78529))
* **blueprint:** add stablisation check method to resource registry ([199f9a4](https://github.com/newstack-cloud/bluelink/commit/199f9a4897c56284989e157530aa96e3fe134ee6))
* **blueprint:** add stub resource for loading placeholder blueprints ([cb08937](https://github.com/newstack-cloud/bluelink/commit/cb089372cfc8b9622adbb6e85bc85b46aba56e81))
* **blueprint:** add support for a blueprint instance name ([e3e43c8](https://github.com/newstack-cloud/bluelink/commit/e3e43c8bc13bd1acafa37bb09c698be9b9fb4a18))
* **blueprint:** add support for complex static values ([e63644f](https://github.com/newstack-cloud/bluelink/commit/e63644f5a9a4419b4fa3fc880633053712851ecc))
* **blueprint:** add support for custom validation for annotation definitions ([9b02f72](https://github.com/newstack-cloud/bluelink/commit/9b02f72d81f006bb4de21ae9bd566b95e2edd7a0))
* **blueprint:** add support for custom validation functions for config fields ([f03c140](https://github.com/newstack-cloud/bluelink/commit/f03c1400e896bc6b049b750dae68fe4bccb3a911))
* **blueprint:** add support for exporting all fields from a data source ([ba6b439](https://github.com/newstack-cloud/bluelink/commit/ba6b439f6de1b5c47230904cb6144b9eec4d7509))
* **blueprint:** add support for ignoring drift for specific resource fields ([1760b83](https://github.com/newstack-cloud/bluelink/commit/1760b83880f418eaa0609c04a06cb35d1e928780))
* **blueprint:** add support for more advanced filter field definitions ([8b74d01](https://github.com/newstack-cloud/bluelink/commit/8b74d013f27c5c0a24eb354c49fc07f1ad13a43a))
* **blueprint:** add support for multiple data source filters ([8d275e3](https://github.com/newstack-cloud/bluelink/commit/8d275e371ea72c4de98b3764886871cc62db3052))
* **blueprint:** add support for none values in substitutions ([c081dd7](https://github.com/newstack-cloud/bluelink/commit/c081dd7c5dbd3e926959179cb82995a125af471e))
* **blueprint:** add support for opting into tracking changes for computed fields ([34018b0](https://github.com/newstack-cloud/bluelink/commit/34018b09c9a3d0ec175a0934590d8bbad149e3e2))
* **blueprint:** add support for parsing none values in substitutions ([9c4ed52](https://github.com/newstack-cloud/bluelink/commit/9c4ed5224f7cd042d98d0f587b23e54df2b504ed))
* **blueprint:** add support for path array selectors ([b045623](https://github.com/newstack-cloud/bluelink/commit/b04562357f9d003a18c6cffc918c84314da3189e))
* **blueprint:** add support for resource locking for link deployment ([0e8ec72](https://github.com/newstack-cloud/bluelink/commit/0e8ec72c32e3b88a0d59fa855903dc3d51be6cb0))
* **blueprint:** add support for runtime binary data for functions ([3cbaaa6](https://github.com/newstack-cloud/bluelink/commit/3cbaaa6dcdbb133cb89c7d34ba5d2dd0c18acb8c))
* **blueprint:** add support for serialising bytes with protobufs ([a16df79](https://github.com/newstack-cloud/bluelink/commit/a16df79125af8c351c8083d7acf1ded3d2f825b1))
* **blueprint:** add the core if function for conditional blueprint elements ([94ac279](https://github.com/newstack-cloud/bluelink/commit/94ac279200972b41666bb5dea148c43ab775c769))
* **blueprint:** add utf8 encode function ([3b8f18d](https://github.com/newstack-cloud/bluelink/commit/3b8f18da2a88ad3ac74185492414ca62f6e8d55a))
* **blueprint:** add utility blueprint cache interface ([e288f85](https://github.com/newstack-cloud/bluelink/commit/e288f8571ecc519bb8a2427a65ec2c77660a1e64))
* **blueprint:** add uuid v4 function ([bfb2fdb](https://github.com/newstack-cloud/bluelink/commit/bfb2fdbc9371e5e241b0ad2ff679cc0652e06bf8))
* **blueprint:** add validation for link annotations ([8c172af](https://github.com/newstack-cloud/bluelink/commit/8c172af3754ba450012e61b22bc2b827248af824))
* **blueprint:** add with call method to function call context ([1bd4fee](https://github.com/newstack-cloud/bluelink/commit/1bd4fee3a46280570d445d9e3fc4ad5a68622bfe))
* **blueprint:** add zap logger adaptor ([524aee4](https://github.com/newstack-cloud/bluelink/commit/524aee4e3390d52848582d351281988bfe2092ea))
* **blueprint:** allow user to override defaults for container dependencies ([faa1fa3](https://github.com/newstack-cloud/bluelink/commit/faa1fa369eea0317540540f70f2c95d675bec887))
* **blueprint:** enhance plugin interfaces with methods to provide rich documentation ([3fa66ac](https://github.com/newstack-cloud/bluelink/commit/3fa66acbbf38d408a1a30a6456f08939abd6787e))
* **blueprint:** ensure instance name is passed to plugin method calls ([64cf760](https://github.com/newstack-cloud/bluelink/commit/64cf76023e8dbfb5315ad3207677c16aff27ab5c))
* **blueprint:** improve key errors to contain rich error context ([13b9736](https://github.com/newstack-cloud/bluelink/commit/13b97366035d0e14dc5f6bb26e0abb265750e459))
* **blueprint:** improve state container interface for resources and links ([8c7fd70](https://github.com/newstack-cloud/bluelink/commit/8c7fd709926cedd2c1fdcf772fdea216a2e14402))
* **blueprint:** integrate drift checks into change staging ([68bc7ef](https://github.com/newstack-cloud/bluelink/commit/68bc7ef292ecf593eef3efa0e5a15fcc884e1246))
* **blueprint:** integrate sensitive logic into field changes ([de0e10b](https://github.com/newstack-cloud/bluelink/commit/de0e10ba926eaf6598b5747bd8c1b5c1088af325))
* **blueprint:** make blueprint schema conversion functions public ([11d405f](https://github.com/newstack-cloud/bluelink/commit/11d405f8e6ffcf50d8796dad0b57b16688067086))
* **blueprint:** make conversion from protobuf mapping node a part of the public interface ([aea1f92](https://github.com/newstack-cloud/bluelink/commit/aea1f927df33f0ad33a4e311a4f5ca64cf996cc1))
* **blueprint:** make converting from source meta to diag range a public helper function ([2456d6f](https://github.com/newstack-cloud/bluelink/commit/2456d6f61d8f9902a315f4beffcc3210891b61d8))
* **blueprint:** make creating a new function call context a part of the public interface ([7644e08](https://github.com/newstack-cloud/bluelink/commit/7644e085550ce898e07b5b084658bc5006350080))
* **blueprint:** make data source conversion from and to protobuf message public ([41c2f8e](https://github.com/newstack-cloud/bluelink/commit/41c2f8ef13fc8ab3313828c94c0c698d5e7c87f4))
* **blueprint:** make link selector conversion a public function ([dd12441](https://github.com/newstack-cloud/bluelink/commit/dd12441286ee1370cb4fcc1a257b45b94c9075c7))
* **blueprint:** simplify resource registry interface for waiting for stability ([4c7c477](https://github.com/newstack-cloud/bluelink/commit/4c7c4775eb68454caf81665ff3d352bd99e5fda7))
* **blueprint:** update protobuf schema and serialisation to support export all field ([9abe247](https://github.com/newstack-cloud/bluelink/commit/9abe247e071fbafdb236195c5887d681fc528b9b))


### Bug Fixes

* **blueprint:** add correction to end columns extracted from json node offsets ([bc93f31](https://github.com/newstack-cloud/bluelink/commit/bc93f3185394b85419ad58314aa25a5a644abfaf))
* **blueprint:** add correction to make sure current link state is passed into link update methods ([6f9682a](https://github.com/newstack-cloud/bluelink/commit/6f9682a9a92ba731829714ea082802e69cac26d2))
* **blueprint:** add correction to mapping node injection behaviour ([da4d4dd](https://github.com/newstack-cloud/bluelink/commit/da4d4ddcc5d53054ef57ee1021b9669e825945f2))
* **blueprint:** add correction to not return error for nullable schema definitions ([1acc5e9](https://github.com/newstack-cloud/bluelink/commit/1acc5e9b6db3d7047baaeeb3aaaffeee8ad6d27b))
* **blueprint:** add corrections to ordering deployment nodes ([784d165](https://github.com/newstack-cloud/bluelink/commit/784d1654cb45ef0cce493b55d842e21a1b10b409))
* **blueprint:** add final fix for empty schema validation checks ([b8a192f](https://github.com/newstack-cloud/bluelink/commit/b8a192f46e8d5e5f6511bff48e4a5dcde2939b47))
* **blueprint:** add fix to avoid panicking when state container is nil ([b52b851](https://github.com/newstack-cloud/bluelink/commit/b52b851cfe643e43d3c2d2baece815c4ea058086))
* **blueprint:** add fix to ensure errors are separated from diags ([f895b92](https://github.com/newstack-cloud/bluelink/commit/f895b92dcceb06800d3d28371731907748e716d6))
* **blueprint:** add fixes to ensure position info is present for jwcc docs ([7f451a2](https://github.com/newstack-cloud/bluelink/commit/7f451a226be675904c6010bd37e00edbb2d52609))
* **blueprint:** add json tags to serialise position fields with lower case field names ([392cc58](https://github.com/newstack-cloud/bluelink/commit/392cc58a922f2d2be98a123263e415ed32a48c20))
* **blueprint:** add missing fields to get external state input ([3a1dec9](https://github.com/newstack-cloud/bluelink/commit/3a1dec9dc6482c45e3d0d0c498bba474f68d8f1b))
* **blueprint:** add missing json tag to end position field ([1127715](https://github.com/newstack-cloud/bluelink/commit/1127715d9ff5e69375a6474ce573b3d3f8f65369))
* **blueprint:** add missing provider context in acquire resource lock input ([58996d9](https://github.com/newstack-cloud/bluelink/commit/58996d9f855270e4ec536103be39e07df2bcc41b))
* **blueprint:** add missing required field for object value type ([6236b1a](https://github.com/newstack-cloud/bluelink/commit/6236b1a91dfa31e1535fdf8f10010655818bcbe5))
* **blueprint:** add missing suggested action for abstract resource type not found ([06820d1](https://github.com/newstack-cloud/bluelink/commit/06820d12f6231d4e67d086a02919aade38fad772))
* **blueprint:** add missing value type checks to plugin config validation ([e3c5b18](https://github.com/newstack-cloud/bluelink/commit/e3c5b18a9ec5cebe064473ddc7c46f614d53bc7c))
* **blueprint:** correct error code for export not found ([ad5497d](https://github.com/newstack-cloud/bluelink/commit/ad5497d86a94b3a79a9ab2d03469be29a9057324))
* **blueprint:** correct json string handling for substitutions ([c8d97b0](https://github.com/newstack-cloud/bluelink/commit/c8d97b0c53041a4cb681abeed327560fdc4faaaa))
* **blueprint:** correct name of transformer context field ([2a3f3ba](https://github.com/newstack-cloud/bluelink/commit/2a3f3ba72db63ecae5cbffe3dc27d40d8c5a5300))
* **blueprint:** correct version constant for blueprint spec final initial version ([d6a90a1](https://github.com/newstack-cloud/bluelink/commit/d6a90a154b66710a06eda63dbd884e83101b6c02))
* **blueprint:** corrects serialisation to treat empty values as nil ([ff31d82](https://github.com/newstack-cloud/bluelink/commit/ff31d823dd16a4f572c11ca8c571cdd9cee680c2))
* **blueprint:** ensure annotations are resolved as scalar types other than strings ([aa67b09](https://github.com/newstack-cloud/bluelink/commit/aa67b09e8535aa71ee157aa85098f9968b0dc43e))
* **blueprint:** escape regexp special chars from dynamic config field name patterns ([fc22a7a](https://github.com/newstack-cloud/bluelink/commit/fc22a7a5661a8619a6d1406449c6eb65c975b583))
* **blueprint:** remove file source registry from function call context ([3614482](https://github.com/newstack-cloud/bluelink/commit/3614482a20c9f6a1bc2f8329f89b438761adc0c2))
* **blueprint:** remove list error action types ([b7a8dbf](https://github.com/newstack-cloud/bluelink/commit/b7a8dbf88d2211e793afba43e3815d0f37c32fd5))
* **blueprint:** remove with call method and make call stack thread-safe ([7347af9](https://github.com/newstack-cloud/bluelink/commit/7347af901b821d931e4ba0fba973ef5873f03822))
* **blueprint:** update common library dependency to version in new org ([c4896ba](https://github.com/newstack-cloud/bluelink/commit/c4896ba49976c6c9adb9989490b18b6b76d51ad2))

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
