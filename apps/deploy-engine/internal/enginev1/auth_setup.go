package enginev1

import (
	"github.com/gorilla/mux"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/core"
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/auth"
	commoncore "github.com/newstack-cloud/bluelink/libs/common/core"
)

func setupAuth(
	config *core.AuthConfig,
	clock commoncore.Clock,
	excludedRoutes []*mux.Route,
) (*auth.Middleware, error) {
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
