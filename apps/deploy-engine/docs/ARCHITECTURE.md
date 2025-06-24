# Architecture

The deploy engine provides the ability to validate and deploy blueprints. It is a combination of a wrapper around the [Blueprint framework](../../blueprint/README.md) and the integration of the [Plugin framework's](../../../libs/plugin-framework) gRPC inter-process plugin system built on top of the `provider.Provider` and `transform.SpecTransformer` interfaces to allow for plugins that can be dynamically loaded at runtime without the security risks of loading arbitrary code into the deploy engine process.
The deploy engine is also bundled with a limited set of blueprint instance state persistence implementations to choose from with configuration.
