package validation

import (
	"context"
	"strings"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

// ValidateBlueprint ensures that the required top-level properties
// of a blueprint are populated.
// (When they are populated the schema takes care of the structure)
func ValidateBlueprint(ctx context.Context, blueprint *schema.Blueprint) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	errors := []error{}

	if blueprint.Version == nil || blueprint.Version.StringValue == nil {
		errors = append(errors, errBlueprintMissingVersion())
	} else {
		isVersionEmpty := strings.TrimSpace(*blueprint.Version.StringValue) == ""
		if isVersionEmpty {
			errors = append(errors, errBlueprintMissingVersion())
		}

		if !isVersionEmpty && !core.SliceContainsComparable(SupportedVersions, *blueprint.Version.StringValue) {
			errors = append(errors, errBlueprintUnsupportedVersion(*blueprint.Version.StringValue))
		}
	}

	if (blueprint.Resources == nil || len(blueprint.Resources.Values) == 0) &&
		(blueprint.Include == nil || len(blueprint.Include.Values) == 0) {
		errors = append(errors, errBlueprintMissingResourcesOrIncludes())
	}

	if len(errors) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errors)
	}

	return diagnostics, nil
}

const (
	// Version2025_11_02 is the version of the blueprint specification
	// that is the sole version of the spec supported by the initial
	// version of the blueprint framework.
	Version2025_11_02 = "2025-11-02"
)

var (
	// SupportedVersions is the list of versions of the blueprint
	// specification that are supported by this version of the blueprint
	// framework.
	SupportedVersions = []string{
		Version2025_11_02,
	}
)
