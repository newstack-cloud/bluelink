# Changelog

## [0.2.0](https://github.com/newstack-cloud/bluelink/compare/blueprint-ls/v0.1.1...blueprint-ls/v0.2.0) (2026-02-01)


### Features

* **blueprint-ls:** add base for new unified model for location-based lsp features ([234f717](https://github.com/newstack-cloud/bluelink/commit/234f71703df095832db75e4067ddb75c8aa44172))
* **blueprint-ls:** add complete link annotation completion suggestions ([d1ee022](https://github.com/newstack-cloud/bluelink/commit/d1ee02212cca8f3ac53b51767c507fa683bcc58f))
* **blueprint-ls:** add completion suggestions and hover for child blueprint exports ([c756d89](https://github.com/newstack-cloud/bluelink/commit/c756d8972847be82ed3768d44a1feada4a57f740))
* **blueprint-ls:** add completion suggestions for link annotations ([3fbc142](https://github.com/newstack-cloud/bluelink/commit/3fbc142935ae4084d57bbcdfd2ebeede2502c4d7))
* **blueprint-ls:** add enum completion suggestions and quick fix code actions ([785c774](https://github.com/newstack-cloud/bluelink/commit/785c7744c4531d4f6f02e1babc0a96985e1f9e04))
* **blueprint-ls:** add find all references for blueprint elements ([8be9349](https://github.com/newstack-cloud/bluelink/commit/8be934997619b37e153ba14a27ce24344240d132))
* **blueprint-ls:** add full blueprint schema field completion for yaml documents ([d669bde](https://github.com/newstack-cloud/bluelink/commit/d669bdebe020edbf19a2e398cfaa618fff5b240b))
* **blueprint-ls:** add go-to definition for export fields, excludes and depends on list ([4993a88](https://github.com/newstack-cloud/bluelink/commit/4993a883530f65ac113c6d98bc82147e7877c9eb))
* **blueprint-ls:** add go-to definitions for local child blueprint includes ([5f0838a](https://github.com/newstack-cloud/bluelink/commit/5f0838a8028553a8f29c74f6ef7b60a45d570ff8))
* **blueprint-ls:** add hover help text for blueprint export field values ([289c135](https://github.com/newstack-cloud/bluelink/commit/289c135308b885c622414bbf2d271f2b2ad4a060))
* **blueprint-ls:** add improvements to diagnostics, completion, hover and symbols ([ac2995d](https://github.com/newstack-cloud/bluelink/commit/ac2995d2abf8de9e89a773e3bc5f2ee303d169d5))
* **blueprint-ls:** add support for configuration to allow users to hide any type warnings ([8f9ea58](https://github.com/newstack-cloud/bluelink/commit/8f9ea58ab2d7af8127045b32651bf115aa875e49))
* **blueprint-ls:** add support for listing matching resources on link selector hover ([4e48fff](https://github.com/newstack-cloud/bluelink/commit/4e48fff3bc03b69baee9c59b5e09c162cef6d8b9))
* **blueprint-ls:** expand hover help support for blueprints ([bf807c9](https://github.com/newstack-cloud/bluelink/commit/bf807c9ffd02a82d0a970a6bebf34ef6b8ce2c2e))
* **blueprint-ls:** implement completion context detection with the new node context ([e997cb2](https://github.com/newstack-cloud/bluelink/commit/e997cb2606e73344e0483f018142aecd31572598))
* **blueprint-ls:** integrate document debouncer to create a smoother experience ([3adf096](https://github.com/newstack-cloud/bluelink/commit/3adf096af7173e121af13f6ab46392471e74a1a5))
* **blueprint-ls:** integrate plugin host into language server ([8a1e59c](https://github.com/newstack-cloud/bluelink/commit/8a1e59ca95a62090802eb1abdb21ad28e5c55494))
* **blueprint-ls:** refactor completion suggestions in subs and export fields ([56ec475](https://github.com/newstack-cloud/bluelink/commit/56ec4756f0dde4498310a88f70c9806a42ab4315))


### Bug Fixes

* **blueprint-ls:** add fixes for type value completions ([6afc829](https://github.com/newstack-cloud/bluelink/commit/6afc82978dec44a1c9f7144a258f5f68c17e0e3c))
* **blueprint-ls:** ensure internal functions are not displayed in completion suggestions ([52170b1](https://github.com/newstack-cloud/bluelink/commit/52170b153bf90a7dd1bdd1d3d130f99cd713a55d))
* **blueprint-ls:** improve error and diagnostic reporting ([a7d3bfd](https://github.com/newstack-cloud/bluelink/commit/a7d3bfda1c4cef03d986fffb4e566d786057eb8a))
* **blueprint-ls:** update ls-builder and remove safe context workaround ([d7a553a](https://github.com/newstack-cloud/bluelink/commit/d7a553a3597e22d7c38c18b6954f476a45dd7b06))


### Dependencies

* **blueprint-ls:** add tree-sitter dependency for partial parsing ([264ea88](https://github.com/newstack-cloud/bluelink/commit/264ea888a91892ca0636943e70c9e4b4517f2a00))
* **blueprint-ls:** bump the go-deps group ([3aa6335](https://github.com/newstack-cloud/bluelink/commit/3aa633506556d578ccb3ea6e2be0be6f85c10eb2))

## [0.1.1](https://github.com/newstack-cloud/bluelink/compare/blueprint-ls/v0.1.0...blueprint-ls/v0.1.1) (2025-12-03)


### Bug Fixes

* **blueprint-ls:** update registry constructor to include new required params ([7825adf](https://github.com/newstack-cloud/bluelink/commit/7825adf64825972c2abab729fe37cdcac4cb9a3e))


### Dependencies

* **blueprint-ls:** update blueprint and common libs ([ff46611](https://github.com/newstack-cloud/bluelink/commit/ff46611212a48e47f22aaacca3cde1288c9f1b01))
