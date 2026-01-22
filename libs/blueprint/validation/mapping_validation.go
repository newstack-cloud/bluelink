package validation

import (
	"context"
	"fmt"

	bpcore "github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/source"
	"github.com/newstack-cloud/bluelink/libs/blueprint/substitutions"
)

// ValidateMappingNode ensures that all the substitutions used in a mapping node
// are valid.
// This is to be used in free form mapping nodes such as `metadata.custom` in data sources
// and resources.
func ValidateMappingNode(
	ctx context.Context,
	usedIn string,
	// Path to attribute in which the mapping node is used
	// in the usedIn element. (e.g. "metadata.custom" used in "datasources.networking")
	attributePath string,
	usedInResourceDerivedFromTemplate bool,
	mappingNode *bpcore.MappingNode,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	if mappingNode == nil || mappingNodeNotSet(mappingNode) {
		return nil, nil
	}

	return validateMappingNode(
		ctx,
		usedIn,
		usedInResourceDerivedFromTemplate,
		attributePath,
		mappingNode,
		mappingNode.SourceMeta,
		/* depth */ 0,
		bpSchema,
		params,
		funcRegistry,
		refChainCollector,
		resourceRegistry,
		dataSourceRegistry,
	)
}

func validateMappingNode(
	ctx context.Context,
	usedIn string,
	usedInResourceDerivedFromTemplate bool,
	attributePath string,
	mappingNode *bpcore.MappingNode,
	wrapperLocation *source.Meta,
	depth int,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}

	if depth > bpcore.MappingNodeMaxTraverseDepth {
		// For performance and security reasons, validation is limited to a pre-determined depth.
		// This should be handled gracefully with no errors and instead reporting
		// info diagnostics to the user.
		rangeEndLocation := getEndLocation(wrapperLocation)
		diagnostics = append(diagnostics, &bpcore.Diagnostic{
			Level: bpcore.DiagnosticLevelInfo,
			Message: fmt.Sprintf(
				"Exceeded max traverse depth of %d. Skipping further validation.",
				bpcore.MappingNodeMaxTraverseDepth,
			),
			Range: bpcore.DiagnosticRangeFromSourceMeta(wrapperLocation, rangeEndLocation),
		})
		return diagnostics, nil
	}

	if mappingNode.Scalar != nil {
		// A scalar value does not need validating as there is no type to validate against
		// for free-form mapping node values.
		return diagnostics, nil
	}

	if mappingNode.StringWithSubstitutions != nil {
		return validateMappingNodeStringWithSubstitutions(
			ctx,
			usedIn,
			attributePath,
			usedInResourceDerivedFromTemplate,
			mappingNode.StringWithSubstitutions,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
		)
	}

	if mappingNode.Fields != nil {
		return validateMappingNodeFields(
			ctx,
			usedIn,
			attributePath,
			usedInResourceDerivedFromTemplate,
			mappingNode.Fields,
			mappingNode.FieldsSourceMeta,
			depth,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
		)
	}

	if mappingNode.Items != nil {
		return validateMappingNodeItems(
			ctx,
			usedIn,
			attributePath,
			usedInResourceDerivedFromTemplate,
			mappingNode.Items,
			depth,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
		)
	}

	return diagnostics, nil
}

func validateMappingNodeFields(
	ctx context.Context,
	usedIn string,
	attributePath string,
	usedInResourceDerivedFromTemplate bool,
	mappingNodeFields map[string]*bpcore.MappingNode,
	mappingNodeFieldsSourceMeta map[string]*source.Meta,
	depth int,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	var errs []error
	for key, field := range mappingNodeFields {
		var wrapperLocation *source.Meta
		if mappingNodeFieldsSourceMeta != nil {
			wrapperLocation = mappingNodeFieldsSourceMeta[key]
		}
		fieldDiagnostics, err := validateMappingNode(
			ctx,
			usedIn,
			usedInResourceDerivedFromTemplate,
			attributePath,
			field,
			wrapperLocation,
			depth+1,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
		)
		if err != nil {
			errs = append(errs, err)
		}
		diagnostics = append(diagnostics, fieldDiagnostics...)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateMappingNodeItems(
	ctx context.Context,
	usedIn string,
	attributePath string,
	usedInResourceDerivedFromTemplate bool,
	mappingNodeItems []*bpcore.MappingNode,
	depth int,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	var errs []error
	for _, item := range mappingNodeItems {
		itemDiagnostics, err := validateMappingNode(
			ctx,
			usedIn,
			usedInResourceDerivedFromTemplate,
			attributePath,
			item,
			/* wrapperLocation */ item.SourceMeta,
			depth+1,
			bpSchema,
			params,
			funcRegistry,
			refChainCollector,
			resourceRegistry,
			dataSourceRegistry,
		)
		if err != nil {
			errs = append(errs, err)
		}
		diagnostics = append(diagnostics, itemDiagnostics...)
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func validateMappingNodeStringWithSubstitutions(
	ctx context.Context,
	usedIn string,
	usedInPropertyPath string,
	usedInResourceDerivedFromTemplate bool,
	stringWithSub *substitutions.StringOrSubstitutions,
	bpSchema *schema.Blueprint,
	params bpcore.BlueprintParams,
	funcRegistry provider.FunctionRegistry,
	refChainCollector refgraph.RefChainCollector,
	resourceRegistry resourcehelpers.Registry,
	dataSourceRegistry provider.DataSourceRegistry,
) ([]*bpcore.Diagnostic, error) {
	diagnostics := []*bpcore.Diagnostic{}
	var errs []error
	for i, stringOrSub := range stringWithSub.Values {
		nextLocation := getSubNextLocation(i, stringWithSub.Values)

		if stringOrSub.SubstitutionValue != nil {
			_, subDiagnostics, err := ValidateSubstitution(
				ctx,
				stringOrSub.SubstitutionValue,
				nextLocation,
				bpSchema,
				usedInResourceDerivedFromTemplate,
				usedIn,
				usedInPropertyPath,
				params,
				funcRegistry,
				refChainCollector,
				resourceRegistry,
				dataSourceRegistry,
			)
			if err != nil {
				errs = append(errs, err)
			} else {
				diagnostics = append(diagnostics, subDiagnostics...)
			}
		}
	}

	if len(errs) > 0 {
		return diagnostics, ErrMultipleValidationErrors(errs)
	}

	return diagnostics, nil
}

func mappingNodeNotSet(mappingNode *bpcore.MappingNode) bool {
	return mappingNode.Scalar == nil && mappingNode.Fields == nil &&
		mappingNode.Items == nil && mappingNode.StringWithSubstitutions == nil
}
