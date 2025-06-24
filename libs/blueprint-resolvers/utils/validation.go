package utils

import (
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/includes"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
)

// ValidateInclude checks if the path is valid and metadata contains all the required fields
// for the given include.
//
// This only checks for string values for the provided required fields.
func ValidateInclude(
	include *subengine.ResolvedInclude,
	includeName string,
	requiredFields []string,
	sourceTypeLabel string,
	resolverName string,
) error {

	path := core.StringValue(include.Path)
	if path == "" {
		return includes.ErrInvalidPath(includeName, resolverName)
	}

	metadata := include.Metadata
	if metadata == nil || metadata.Fields == nil {
		return includes.ErrInvalidMetadata(
			includeName,
			fmt.Sprintf("invalid metadata provided for the %s include", sourceTypeLabel),
		)
	}

	for _, field := range requiredFields {
		fieldValue := core.StringValue(metadata.Fields[field])
		if fieldValue == "" {
			return includes.ErrInvalidMetadata(
				includeName,
				fmt.Sprintf(
					"missing %s field in metadata for the %s include",
					field,
					sourceTypeLabel,
				),
			)
		}
	}

	return nil
}
