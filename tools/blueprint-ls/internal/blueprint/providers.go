package blueprint

import (
	"context"

	"github.com/two-hundred/celerity/libs/blueprint/provider"
	"github.com/two-hundred/celerity/libs/blueprint/providerhelpers"
)

// LoadProviders deals with loading the providers to be used for validating
// and in providing other LSP features for blueprint such as function signatures,
// hover information and completion items.
//
// The language server uses the build engine plugin system to load gRPC provider
// plugins.
func LoadProviders(ctx context.Context) (map[string]provider.Provider, error) {

	// We won't be calling any functions in the language server, so we can provide
	// nil values for all the parameters that are only used at runtime in function
	// calls.
	coreProvider := providerhelpers.NewCoreProvider(
		/* linkStateRetriever */ nil,
		/* blueprintInstanceIDRetriever */ nil,
		/* resolveWorkingDir */ nil,
		/* clock */ nil,
	)

	// Purely for testing purposes, should be removed once some actual
	// providers have been implemented to test with.
	celerityProvider := NewCelerityProvider()

	// TODO: load provider plugins through the build engine plugin system

	return map[string]provider.Provider{
		"core":     coreProvider,
		"celerity": celerityProvider,
	}, nil
}
