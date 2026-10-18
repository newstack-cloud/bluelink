package enginev1

import (
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/auth"
	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

// The shortest API key that an audit record's fingerprint does not weaken.
const minRecommendedAPIKeyLength = 32

func setupAuth(
	config *core.AuthConfig,
	clock commoncore.Clock,
	excludedRoutes []*mux.Route,
	logger bpcore.Logger,
) (*auth.Middleware, error) {
	warnAboutWeakAPIKeys(config, logger)

	authCheckers := []auth.Checker{}
	jwtAuthChecker, err := auth.LoadJWTService(config)
	if err != nil {
		if len(config.APIKeys) == 0 &&
			len(config.BluelinkSigV1KeyPairs) == 0 {
			// There must be at least one auth method configured.
			return nil, err
		}
		// If JWT auth is not configured, we can still
		// use the other auth methods, so there is no need
		// to return an error early.
	} else {
		authCheckers = append(authCheckers, jwtAuthChecker)
	}

	authCheckers = append(
		authCheckers,
		auth.NewSigV1Service(
			config.BluelinkSigV1KeyPairs,
			clock,
			/* options */ nil,
		),
		auth.NewAPIKeyService(config),
	)

	authMiddleware, err := auth.NewMiddleware(
		auth.NewMultiAuthChecker(
			authCheckers...,
		),
		excludedRoutes,
	)
	if err != nil {
		return nil, err
	}

	return authMiddleware, nil
}

// Reports configured API keys short enough that an audit record's fingerprint of them is
// worth guessing against.
//
// Counted rather than named, so that the warning does not say which key is weak, the
// warning goes to the same logs the fingerprints do.
func warnAboutWeakAPIKeys(config *core.AuthConfig, logger bpcore.Logger) {
	weak := 0
	for _, apiKey := range config.APIKeys {
		if len(apiKey) < minRecommendedAPIKeyLength {
			weak += 1
		}
	}

	if weak == 0 {
		return
	}

	logger.Warn(
		"one or more configured API keys are shorter than the recommended length, and an "+
			"audit record identifies the key that acted by a digest of it, so a short or "+
			"memorable key can be recovered from the logs by guessing; use keys of at "+
			"least the recommended length generated from a random source",
		bpcore.IntegerLogField("weakKeyCount", int64(weak)),
		bpcore.IntegerLogField("recommendedLength", int64(minRecommendedAPIKeyLength)),
	)
}
