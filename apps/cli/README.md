# Bluelink CLI

[![Coverage](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-cli&metric=coverage)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-cli)
[![Security Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-cli&metric=security_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-cli)
[![Maintainability Rating](https://sonarcloud.io/api/project_badges/measure?project=newstack-cloud_bluelink-cli&metric=sqale_rating)](https://sonarcloud.io/summary/new_code?id=newstack-cloud_bluelink-cli)

The CLI for managing blueprints used for Infrastructure as Code along with managing Bluelink provider and transformer plugins.

The CLI provides the following main features:

- **Initialise projects**: Create a new blueprint project from a template.
- **Validate blueprints**: Validate a Bluelink blueprint, ensuring the blueprint is well-formed and meets the requirements of resource providers.
- **Deploy blueprints**: Deploy a blueprint.
- **Manage plugins**: Install, update, and remove Provider and Transformer plugins for the Deploy Engine running on the same machine as the CLI.

## Additional documentation

- [Contributing](docs/CONTRIBUTING.md)
- [Architecture](docs/ARCHITECTURE.md)
