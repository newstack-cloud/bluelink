package subengine

import (
	"fmt"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

type resolveOnDeployError struct {
	propertyPath string
}

func (e resolveOnDeployError) Error() string {
	return "property " + e.propertyPath + " can not be resolved during change staging," +
		" it can only be resolved during deployment"
}

func errMustResolveOnDeploy(elementName string, elementProp string) error {
	return &resolveOnDeployError{
		propertyPath: bpcore.ElementPropertyPath(elementName, elementProp),
	}
}

type resolveOnDeployErrors struct {
	errors []*resolveOnDeployError
}

func (e resolveOnDeployErrors) Error() string {
	return fmt.Sprintf(
		"multiple properties can not be resolved during change staging,"+
			" they can only be resolved during deployment (%d errors)",
		len(e.errors),
	)
}

func errMustResolveOnDeployMultiple(
	paths []string,
) error {
	errors := make([]*resolveOnDeployError, len(paths))
	for i, path := range paths {
		errors[i] = &resolveOnDeployError{
			propertyPath: path,
		}
	}
	return &resolveOnDeployErrors{
		errors: errors,
	}
}
