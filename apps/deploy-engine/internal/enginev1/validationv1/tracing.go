package validationv1

import (
	"errors"
)

// The span name for the validation this controller carries on with after the request
// that asked for it has been answered.
const validationSpanName = "validation.run"

// Why a validation that reported a failed status failed.
//
// A validation fails either on an error or on an error diagnostic in the blueprint, and
// only the first carries an error to record, so the second is described rather than left
// as a failure with nothing said about it.
func validationFailedError(err error) error {
	if err != nil {
		return err
	}

	return errors.New("the blueprint has at least one error diagnostic")
}
