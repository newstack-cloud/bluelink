package engine

import (
	"errors"
	"fmt"

	deerrors "github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
	"go.uber.org/zap"
)

// SimplifyError deals with simplifying specific deploy engine errors
// that are a part of the deploy engine client library API and
// transforming them into something easier to digest for the user.
// When an error is simplified, the original error will be logged at the debug level with
// the provided logger so there is a traceable record of the original error when debugging.
func SimplifyError(err error, logger *zap.Logger) error {
	authPrepErr := &deerrors.AuthPrepError{}
	if errors.As(err, &authPrepErr) {
		logger.Debug("auth prep error", zap.Error(err))
		return fmt.Errorf(
			"failed to prepare authentication headers for the deploy engine, please make sure \n" +
				"at least one of the supported authentication methods is configured for the CLI",
		)
	}
	return err
}
