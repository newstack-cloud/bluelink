package deployengine

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/newstack-cloud/bluelink/libs/deploy-engine-client/errors"
)

func createAuthPrepError(message string) *errors.AuthPrepError {
	return &errors.AuthPrepError{
		Message: message,
	}
}

func createAuthInitError(message string) *errors.AuthInitError {
	return &errors.AuthInitError{
		Message: message,
	}
}

func createSerialiseError(message string) *errors.SerialiseError {
	return &errors.SerialiseError{
		Message: message,
	}
}

func createDeserialiseError(message string) *errors.DeserialiseError {
	return &errors.DeserialiseError{
		Message: message,
	}
}

func createRequestPrepError(message string) *errors.RequestPrepError {
	return &errors.RequestPrepError{
		Message: message,
	}
}

func createRequestError(err error) *errors.RequestError {
	return &errors.RequestError{
		Err: err,
	}
}

func createClientError(resp *http.Response) *errors.ClientError {
	errResp := &errors.Response{}
	errRespBytes, _ := io.ReadAll(resp.Body)
	json.Unmarshal(errRespBytes, errResp)
	if errResp.Message == "" {
		errResp.Message = fmt.Sprintf(
			"client error: %s",
			resp.Status,
		)
	}

	return &errors.ClientError{
		StatusCode:            resp.StatusCode,
		Message:               errResp.Message,
		ValidationErrors:      errResp.Errors,
		ValidationDiagnostics: errResp.Diagnostics,
	}
}
