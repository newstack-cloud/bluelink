# Architecture Overview

Bluelink is a collection of tools for defining and deploying infrastructure in a declarative manner based around an _"infrastructure as relationships"_ model, where configuration such as permissions and network access is determined by stating relationships between resources via [links](https://www.bluelink.dev/docs/bluelink/blueprints/links/). It is designed to be extensible and can be used to deploy resources in any environment, including cloud providers, on-premise environments and local development environments.

The following diagram provides an overview of the components that make up Bluelink and how they interact with each other:

![Bluelink Architecture Overview](/resources/bluelink-architecture-overview.png)
