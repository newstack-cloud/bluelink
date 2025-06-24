# blueprint framework

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-blueprint&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-blueprint)

The blueprint framework is the foundation for blueprint deployments managed by the [deploy engine](../deploy-engine), it can also be used as a standalone framework for cloud and other resource deployment systems.

The framework is made up of a collection of modules containing orchestration functionality for planning and deploying blueprints along with a collection of interfaces for resource provider plugins, state persistence, and other useful components in the framework's model. See the [architecture](docs/ARCHITECTURE.md) or the [blueprint framework](https://www.bluelink.dev/blueprint-framework/docs/intro) docs for more information.

This is an implementation of the [Blueprint Specification](https://bluelink.dev/docs/blueprint/specification).

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
- [Architecture](docs/ARCHITECTURE.md)
