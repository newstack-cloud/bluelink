# Changelog

## 1.0.0 (2025-12-02)


### Features

* **deploy-engine:** add authentication implementation ([953db2f](https://github.com/newstack-cloud/bluelink/commit/953db2fcd574a1f9952a34b31eacdcfe80711f90))
* **deploy-engine:** add functionality to validate plugin configuration ([2955935](https://github.com/newstack-cloud/bluelink/commit/29559355915f9b1eb00c146f647d6059603d229c))
* **deploy-engine:** add initial implementation of deploy engine ([f9d000a](https://github.com/newstack-cloud/bluelink/commit/f9d000a9cf28850c207f6beeb734b5462ef1b362))
* **deploy-engine:** add support for dependencies in config object ([d5dbcb8](https://github.com/newstack-cloud/bluelink/commit/d5dbcb82f49794d54c685305e26fbe91aa3a79cc))
* **deploy-engine:** add support for loading configuration from files ([77a2e3b](https://github.com/newstack-cloud/bluelink/commit/77a2e3bc873a0bc7db358759c6bd69c04ff0c78a))
* **deploy-engine:** integrate plugin config validation into change staging and deploy endpoints ([36a98e1](https://github.com/newstack-cloud/bluelink/commit/36a98e122619f8d5c609ef77d2592f77bb0a286a))
* **deploy-engine:** make resolver s3 path style configurable ([2624775](https://github.com/newstack-cloud/bluelink/commit/2624775d4edf155099b389b975a2bc52466562eb))
* **plugin-framework:** add support for adding env vars for os cmd executor ([6738039](https://github.com/newstack-cloud/bluelink/commit/67380392368a2c812457201f2c48ed8048a9d000))


### Bug Fixes

* **deploy-engine:** add end of stream marker to serialised event data ([36a31c0](https://github.com/newstack-cloud/bluelink/commit/36a31c0c1cc51abb1ce5c4e904f339a3f1709743))
* **deploy-engine:** add missing state container dependency for resource registry ([cfda28f](https://github.com/newstack-cloud/bluelink/commit/cfda28fbc705f0bfdb963df5997e7ebf803a6a91))
* **deploy-engine:** correct dependencies to versions in new org ([be1706a](https://github.com/newstack-cloud/bluelink/commit/be1706afad034e6fda4aa2f72bb632a95935f2cd))
* **deploy-engine:** correct method name for event clean up ([2394dad](https://github.com/newstack-cloud/bluelink/commit/2394dadbe16ef640efe40812c294f89edd1f6f4a))
* **deploy-engine:** update plugin framework and pass in registry for resource lookup ([211bd20](https://github.com/newstack-cloud/bluelink/commit/211bd20345e1b24217282b4a85c6a4bb459078d5))
