package handlers

import (
	"context"
	"fmt"
	"io"

	"github.com/newstack-cloud/bluelink/apps/cli/internal/engine"
	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/types"
	"go.uber.org/zap"
)

// NewValidateHandler creates a new validation handler
// for non-interactive environments.
func NewValidateHandler(
	deployEngine engine.DeployEngine,
	blueprintFile string,
	writer io.Writer,
	logger *zap.Logger,
) Handler {
	return HandlerFunc(func(ctx context.Context) error {
		fmt.Fprintf(writer, "Validating blueprint file: %s\n", blueprintFile)
		blueprintValidation, err := deployEngine.CreateBlueprintValidation(
			ctx,
			&types.CreateBlueprintValidationPayload{
				BlueprintDocumentInfo: types.BlueprintDocumentInfo{},
			},
			&types.CreateBlueprintValidationQuery{},
		)
		if err != nil {
			return engine.SimplifyError(err, logger)
		}

		streamTo := make(chan types.BlueprintValidationEvent)
		errChan := make(chan error)
		err = deployEngine.StreamBlueprintValidationEvents(
			ctx,
			blueprintValidation.ID,
			streamTo,
			errChan,
		)
		if err != nil {
			return err
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case err := <-errChan:
				if err != nil {
					return err
				}
			case event, open := <-streamTo:
				if !open {
					fmt.Fprintln(writer, "Validation stream closed")
					return nil
				}
				fmt.Fprintf(writer, "Received event: %+v\n", event)
			}
		}
	})
}
