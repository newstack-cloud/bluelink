# Changelog

## [0.2.0](https://github.com/newstack-cloud/bluelink/compare/cli/v0.1.1...cli/v0.2.0) (2026-02-01)


### Features

* **cli:** add cleanup command ([14c9a6a](https://github.com/newstack-cloud/bluelink/commit/14c9a6ae5a659252a6ce208902918b04db59fce7))
* **cli:** add command for listing plugins with dependency trees ([e82bc1d](https://github.com/newstack-cloud/bluelink/commit/e82bc1deeefebe569183a75c202eaef3a9bf591d))
* **cli:** add command to export blueprint instances ([df4a34f](https://github.com/newstack-cloud/bluelink/commit/df4a34f866ce6944290f89dd588aadca2c376672))
* **cli:** add command to import instance state from a json file ([e2a3ee7](https://github.com/newstack-cloud/bluelink/commit/e2a3ee740bfcc77339a8dd3adaf0597d80612989))
* **cli:** add command to inspect a blueprint instance ([b2a4c29](https://github.com/newstack-cloud/bluelink/commit/b2a4c29fb692433f22606b50e866cbf3b7ba8cce))
* **cli:** add command to list and search blueprint instances ([9e9a76b](https://github.com/newstack-cloud/bluelink/commit/9e9a76b5c69d7969d1928aff2e2bc852160f0657))
* **cli:** add command to list and search blueprint project templates ([b4c2f44](https://github.com/newstack-cloud/bluelink/commit/b4c2f448a2f7bbd08f6dab4627efdc8e27bd7d9e))
* **cli:** add commands for installing plugins and registry login ([023a367](https://github.com/newstack-cloud/bluelink/commit/023a367ccb64a49934442753e0852b70fb99872a))
* **cli:** add complete implementation of init command ([21ffb36](https://github.com/newstack-cloud/bluelink/commit/21ffb361578f6214e4656e4f41cfb0386a50156e))
* **cli:** add feature complete deploy command ([1bd4f02](https://github.com/newstack-cloud/bluelink/commit/1bd4f02a527be48109c347277eb67c17e7049f5a))
* **cli:** add feature complete stage command ([ccc6460](https://github.com/newstack-cloud/bluelink/commit/ccc6460032395669427ed57968ff6d0b0ade8ae0))
* **cli:** add fully functioning destroy command ([93849b5](https://github.com/newstack-cloud/bluelink/commit/93849b5b098d9cbdf9cbd43a3b15a683708e5e9b))
* **cli:** add plugins uninstall command ([e61ad11](https://github.com/newstack-cloud/bluelink/commit/e61ad11168a3a0b00bf8af3d4d6fad4bac0df011))
* **cli:** add preflight plugin check and install for main commands ([03f4b5b](https://github.com/newstack-cloud/bluelink/commit/03f4b5b3a7380424e278285d8b84590b0c4d2e61))
* **cli:** add stage command for change staging ([8c118ae](https://github.com/newstack-cloud/bluelink/commit/8c118aed001cc1f802076c0180e93c501460a9ed))


### Bug Fixes

* **blueprint:** improve failure handling for deployment orchestration ([3bedabc](https://github.com/newstack-cloud/bluelink/commit/3bedabcc1107edb05f1a71b3b4e2e9a1f974bd98))
* **cli:** add correction to not found json output for export command ([f5a318e](https://github.com/newstack-cloud/bluelink/commit/f5a318e3323f454ed3785c35d95d92273d468234))
* **cli:** add fixes for passing through blueprint files from remote sources ([2a0db3f](https://github.com/newstack-cloud/bluelink/commit/2a0db3fc0a885dbb2308ae27f84272a3fedc0674))
* **cli:** add fixes for plugins login and install commands ([d80a07e](https://github.com/newstack-cloud/bluelink/commit/d80a07e36643fd39a7cfe2186afed82cd131dcb5))
* **cli:** add improvements to error reporting ([07674b5](https://github.com/newstack-cloud/bluelink/commit/07674b56b2cf0927ea2482fad4065aab9728c640))
* **cli:** add missing options in interactive prompt form for the stage command ([e43cb30](https://github.com/newstack-cloud/bluelink/commit/e43cb3002ed42e34892098ba085e1c6030a5f20e))
* **cli:** add security fix for tar extraction to protect symlinks ([144f99e](https://github.com/newstack-cloud/bluelink/commit/144f99eebd176b634a1a53219e429811007d416b))
* **cli:** address security hotspot by using absolute path for git binary ([5a96d96](https://github.com/newstack-cloud/bluelink/commit/5a96d964202855feec7ea160320fcdb66bc6e866))
* **cli:** ensure absolute unwritable dirs are used for oauth2 code flow commands ([2939db4](https://github.com/newstack-cloud/bluelink/commit/2939db4602327c407a83d90aec4ae3c033accf01))


### Dependencies

* **cli:** bump the go-deps group across 1 directory with 10 updates ([02f5def](https://github.com/newstack-cloud/bluelink/commit/02f5defec56f9080b6c7efe4d3da9dc55a720aa8))
* **cli:** bump the go-deps group in /apps/cli with 8 updates ([0719ebf](https://github.com/newstack-cloud/bluelink/commit/0719ebfaa6732ea62acfac2dad5712c019b5da59))
* **cli:** remove replace directive and update deploy-cli-sdk ([bec51f0](https://github.com/newstack-cloud/bluelink/commit/bec51f0639d82bb36c823f4fa1d855ed7c31fe39))
* **cli:** update bluelink libs ([e9e3ccc](https://github.com/newstack-cloud/bluelink/commit/e9e3ccc259936852242d2f7ac2c409f553fa75c1))

## [0.1.1](https://github.com/newstack-cloud/bluelink/compare/cli/v0.1.0...cli/v0.1.1) (2025-12-03)


### Features

* **cli:** add dynamic version info to cli version command ([71337cd](https://github.com/newstack-cloud/bluelink/commit/71337cd0440ee48f7d3dabfc43dc5b54b6c725e1))


### Dependencies

* **cli:** update blueprint and state libs ([3c94cc0](https://github.com/newstack-cloud/bluelink/commit/3c94cc0dfde03987fcb53ec8a8921d4d8c8216de))
