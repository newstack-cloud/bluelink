package utils

import "github.com/newstack-cloud/bluelink/apps/deploy-engine/errors"

func errUnsupportedBlueprintFormat(fileName string) error {
	return &errors.DeployEngineError{
		Message: "unsupported blueprint format file \"" + fileName +
			"\", only json or yaml files with extensions are supported",
	}
}

const (
	// UnexpectedErrorMessage is a generic error message for unexpected errors
	// that occur in the deploy engine.
	UnexpectedErrorMessage = "an unexpected error occurred"
)
