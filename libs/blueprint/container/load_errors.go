package container

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"github.com/newstack-cloud/bluelink/libs/common/core"
)

const (
	// ErrorReasonCodeInvalidSpecExtension is provided
	// when the reason for a blueprint spec load error
	// is due to an invalid specification file extension.
	ErrorReasonCodeInvalidSpecExtension errors.ErrorReasonCode = "invalid_spec_ext"
	// ErrorReasonCodeInvalidResourceType is provided
	// when the reason for a blueprint spec load error
	// is due to an invalid resource type provided in one
	// of the resources in the spec.
	ErrorReasonCodeInvalidResourceType errors.ErrorReasonCode = "invalid_resource_type"
	// ErrorReasonCodeMissingResource is provided when the
	// reason for a blueprint spec load error is due to
	// the resource provider missing an implementation for the
	// resource type for one of the resources in the spec.
	ErrorReasonCodeMissingResource errors.ErrorReasonCode = "missing_resource"
	// ErrorReasonCodeResourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more resources in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeResourceValidationErrors errors.ErrorReasonCode = "resource_validation_errors"
	// ErrorReasonCodeDataSourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more data sources in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeDataSourceValidationErrors errors.ErrorReasonCode = "data_source_validation_errors"
	// ErrorReasonCodeResourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more variables in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeVariableValidationErrors errors.ErrorReasonCode = "variable_validation_errors"
	// ErrorReasonCodeIncludeValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more includes in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeIncludeValidationErrors errors.ErrorReasonCode = "include_validation_errors"
	// ErrorReasonCodeResourceValidationErrors is provided
	// when the reason for a blueprint spec load error is due to
	// a collection of errors for one or more variables in the spec.
	// This should be used for a wrapper error that holds more specific
	// errors which can be used for reporting useful information
	// about issues with the spec.
	ErrorReasonCodeExportValidationErrors errors.ErrorReasonCode = "export_validation_errors"
	// ErrorReasonMissingTransformers is provided when the
	// reason for a blueprint spec load error is due to a spec referencing
	// transformers that aren't supported by the blueprint loader
	// used to parse the schema.
	ErrorReasonMissingTransformers errors.ErrorReasonCode = "missing_transformers"
	// ErrorReasonCodeTransformValidationErrors is provided when one or
	// more transformers produced error-level diagnostics during the
	// transform/emit phase. Promoting these to load errors prevents the
	// deploy engine from acting on a partially expanded blueprint when a
	// transformer signals a fatal user-input problem (for example, an
	// unsupported runtime) via diagnostics rather than a Go error.
	ErrorReasonCodeTransformValidationErrors errors.ErrorReasonCode = "transform_validation_errors"
)

func errUnsupportedSpecFileExtension(filePath string) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeInvalidSpecExtension,
		Err:        fmt.Errorf("unsupported spec file extension in %s, only json and yaml are supported", filePath),
	}
}

func errVariableValidationError(errorMap map[string][]error) error {
	errs := flattenErrorMap(errorMap)
	errCount := len(errs)

	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeVariableValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d variables in the spec", errCount),
		ChildErrors: errs,
	}
}

func errResourceValidationError(errorMap map[string][]error) error {
	errs := flattenErrorMap(errorMap)
	errCount := len(errs)

	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeResourceValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d resources in the spec", errCount),
		ChildErrors: errs,
	}
}

func errDataSourceValidationError(errorMap map[string][]error) error {
	errs := flattenErrorMap(errorMap)
	errCount := len(errs)

	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeDataSourceValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d data sources in the spec", errCount),
		ChildErrors: errs,
	}
}

func errIncludeValidationError(errorMap map[string]error) error {
	errCount := len(errorMap)
	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeIncludeValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d includes in the spec", errCount),
		ChildErrors: core.MapToSlice(errorMap),
	}
}

func errExportValidationError(errorMap map[string]error) error {
	errCount := len(errorMap)
	return &errors.LoadError{
		ReasonCode:  ErrorReasonCodeExportValidationErrors,
		Err:         fmt.Errorf("validation failed due to issues with %d exports in the spec", errCount),
		ChildErrors: core.MapToSlice(errorMap),
	}
}

// Link errors carry the resources involved in the link, use them to attribute the
// failure to a location in the source blueprint document.
// Without this, the error is reported without a location and tools that surface
// diagnostics (e.g. the language server) fall back to the start of the document.
func positionedLinkError(err error) error {
	linkErr, isLinkErr := err.(*links.LinkError)
	if !isLinkErr {
		return err
	}

	fromResource, toResource := linkErrorResources(linkErr)
	if fromResource == nil {
		return positionedLinkErrorChildren(linkErr)
	}

	sourceMetas := linkErrorSourceMetas(linkErr, fromResource, toResource)
	if len(sourceMetas) == 1 {
		return linkErrorAtSourceMeta(linkErr, sourceMetas[0])
	}

	childErrors := []error{}
	for _, sourceMeta := range sourceMetas {
		childErrors = append(childErrors, linkErrorAtSourceMeta(linkErr, sourceMeta))
	}

	return &errors.LoadError{
		ReasonCode:  errors.ErrorReasonCode(linkErr.ReasonCode),
		Err:         linkErr.Err,
		ChildErrors: childErrors,
	}
}

// Wrappers such as the error reported for a set of circular links hold an error
// for each link, position each one against the resources that declare that link.
func positionedLinkErrorChildren(linkErr *links.LinkError) error {
	if len(linkErr.ChildErrors) == 0 {
		return linkErr
	}

	childErrors := core.Map(
		linkErr.ChildErrors,
		func(childErr error, _ int) error {
			return positionedLinkError(childErr)
		},
	)

	return &errors.LoadError{
		ReasonCode:  errors.ErrorReasonCode(linkErr.ReasonCode),
		Err:         linkErr.Err,
		ChildErrors: childErrors,
	}
}

func linkErrorResources(linkErr *links.LinkError) (*schema.Resource, *schema.Resource) {
	if linkErr.FromResource != nil {
		return linkErr.FromResource.Resource, selectedResourceSchema(linkErr.ToResource)
	}

	if linkErr.FromLink != nil {
		return linkErr.FromLink.Resource, chainLinkResourceSchema(linkErr.ToLink)
	}

	return nil, nil
}

func selectedResourceSchema(resource *links.ResourceWithNameAndSelectors) *schema.Resource {
	if resource == nil {
		return nil
	}

	return resource.Resource
}

func chainLinkResourceSchema(link *links.ChainLinkNode) *schema.Resource {
	if link == nil {
		return nil
	}

	return link.Resource
}

func linkErrorSourceMetas(
	linkErr *links.LinkError,
	fromResource *schema.Resource,
	toResource *schema.Resource,
) []*source.Meta {
	if toResource == nil {
		return []*source.Meta{resourceTypeSourceMeta(fromResource)}
	}

	selectorSourceMeta, labelSourceMeta := linkSelectionSourceMeta(
		fromResource,
		toResource,
	)

	sourceMetas := []*source.Meta{}
	if selectorSourceMeta != nil {
		sourceMetas = append(sourceMetas, selectorSourceMeta)
	}
	if labelSourceMeta != nil {
		sourceMetas = append(sourceMetas, labelSourceMeta)
	}

	if len(sourceMetas) == 0 {
		return []*source.Meta{resourceTypeSourceMeta(linkErr.FromResource.Resource)}
	}

	return sourceMetas
}

func linkSelectionSourceMeta(
	selectorResource *schema.Resource,
	selectedResource *schema.Resource,
) (*source.Meta, *source.Meta) {
	selectorLabels := linkSelectorLabels(selectorResource)
	labels := resourceLabels(selectedResource)
	if selectorLabels == nil || labels == nil {
		return nil, nil
	}

	for _, labelKey := range slices.Sorted(maps.Keys(selectorLabels.Values)) {
		if labels.Values[labelKey] != selectorLabels.Values[labelKey] {
			continue
		}

		return selectorLabels.SourceMeta[labelKey], labels.SourceMeta[labelKey]
	}

	return nil, nil
}

func linkErrorAtSourceMeta(linkErr *links.LinkError, sourceMeta *source.Meta) error {
	posRange := source.PositionRangeFromSourceMeta(sourceMeta)

	return &errors.LoadError{
		ReasonCode:     errors.ErrorReasonCode(linkErr.ReasonCode),
		Err:            linkErr.Err,
		Line:           posRange.Line,
		EndLine:        posRange.EndLine,
		Column:         posRange.Column,
		EndColumn:      posRange.EndColumn,
		ColumnAccuracy: posRange.ColumnAccuracy,
	}
}

func resourceTypeSourceMeta(resource *schema.Resource) *source.Meta {
	if resource == nil || resource.Type == nil {
		return nil
	}

	return resource.Type.SourceMeta
}

func linkSelectorLabels(resource *schema.Resource) *schema.StringMap {
	if resource == nil || resource.LinkSelector == nil {
		return nil
	}

	return resource.LinkSelector.ByLabel
}

func resourceLabels(resource *schema.Resource) *schema.StringMap {
	if resource == nil || resource.Metadata == nil {
		return nil
	}

	return resource.Metadata.Labels
}

func errTransformersMissing(missingTransformers []string, childErrors []error, line *int, column *int) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonMissingTransformers,
		Err: fmt.Errorf(
			"the following transformers are missing in the blueprint loader: %s", strings.Join(missingTransformers, ", "),
		),
		ChildErrors: childErrors,
		Line:        line,
		Column:      column,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryTransformer,
			ReasonCode: ErrorReasonMissingTransformers,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallTransformers),
					Title:       "Install Transformers",
					Description: "Install the missing transformers",
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckTransformers),
					Title:       "Check Transformers",
					Description: "Explore the available transformers",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"missingTransformers": missingTransformers,
			},
		},
	}
}

func errTransformerMissing(transformer string, line *int, column *int) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonMissingTransformers,
		Err: fmt.Errorf(
			"the following transformer is missing from the blueprint loader: %s", transformer,
		),
		Line:   line,
		Column: column,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryTransformer,
			ReasonCode: ErrorReasonMissingTransformers,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallTransformer),
					Title:       "Install Transformer",
					Description: "Install the missing transformer",
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckTransformers),
					Title:       "Check Transformers",
					Description: "Explore the available transformers",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"missingTransformer": transformer,
			},
		},
	}
}

func errMissingVariableType(variableName string, location *source.Meta) error {
	line, column := source.PositionFromSourceMeta(location)
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValidationErrors,
		Err: fmt.Errorf(
			"variable type missing for variable %s", variableName,
		),
		Line:   line,
		Column: column,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableValidationErrors,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeAddVariableType),
					Title:       "Add Variable Type",
					Description: "Add the missing variable type",
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckVariableType),
					Title:       "Check Variable Type",
					Description: "Explore the available variable types",
					Priority:    2,
				},
			},
			Metadata: map[string]any{
				"variableName": variableName,
			},
		},
	}
}

func errInvalidCustomVariableType(
	variableName string,
	variableType schema.VariableType,
	availableTypes []string,
	line *int,
	column *int,
) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValidationErrors,
		Err: fmt.Errorf(
			"invalid custom variable type %s for variable %s", variableType, variableName,
		),
		Line:   line,
		Column: column,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableValidationErrors,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallProvider),
					Title:       "Install Provider for Variable Type",
					Description: "Install the provider for the variable type",
					Priority:    1,
				},
				{
					Type:        string(errors.ActionTypeCheckVariableType),
					Title:       "Check Variable Type",
					Description: "Explore the available variable types",
					Priority:    2,
				},
			},
			Metadata: validation.AddSuggestionsToMetadata(
				map[string]any{
					"variableName":      variableName,
					"variableType":      variableType,
					"providerNamespace": provider.ExtractProviderFromItemType(string(variableType)),
				},
				string(variableType),
				availableTypes,
			),
		},
	}
}

func errMissingProviderForCustomVarType(
	providerKey string,
	variableName string,
	variableType schema.VariableType,
	line *int,
	column *int,
) error {
	return &errors.LoadError{
		ReasonCode: ErrorReasonCodeVariableValidationErrors,
		Err: fmt.Errorf(
			"missing provider %s for custom variable type %s in variable %s", providerKey, variableType, variableName,
		),
		Line:   line,
		Column: column,
		Context: &errors.ErrorContext{
			Category:   errors.ErrorCategoryVariableType,
			ReasonCode: ErrorReasonCodeVariableValidationErrors,
			SuggestedActions: []errors.SuggestedAction{
				{
					Type:        string(errors.ActionTypeInstallProvider),
					Title:       "Install Provider for Variable Type",
					Description: "Install the provider for the variable type",
					Priority:    1,
				},
			},
			Metadata: map[string]any{
				"variableName":      variableName,
				"variableType":      variableType,
				"providerNamespace": providerKey,
			},
		},
	}
}

func flattenErrorMap(errorMap map[string][]error) []error {
	errs := []error{}
	for _, errSlice := range errorMap {
		errs = append(errs, errSlice...)
	}
	return errs
}
