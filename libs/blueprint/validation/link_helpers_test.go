package validation

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

type testResourceTypeAResourceTypeBLink struct{}

func (l *testResourceTypeAResourceTypeBLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource: provider.LinkPriorityResourceNone,
	}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{
			"test/resourceTypeA::test.string.annotation": {
				Name:        "test.string.annotation",
				Label:       "Test String Annotation",
				Type:        core.ScalarTypeString,
				Description: "This is a test string annotation for resource type A.",
				AllowedValues: []*core.ScalarValue{
					core.ScalarFromString("test-value"),
					core.ScalarFromString("targeted-test-value"),
				},
				Required:  true,
				AppliesTo: provider.LinkAnnotationResourceA,
			},
			"test/resourceTypeA::test.string.<resourceTypeBName>.annotation": {
				Name:        "test.string.<resourceTypeBName>.annotation",
				Label:       "Test String Annotation for Resource Type B",
				Type:        core.ScalarTypeString,
				Description: "This is a test string annotation for resource type A that targets resource type B.",
				AppliesTo:   provider.LinkAnnotationResourceA,
			},
			"test/resourceTypeA::test.bool.<resourceTypeBName>.annotation": {
				Name:        "test.bool.<resourceTypeBName>.annotation",
				Label:       "Test Boolean Annotation for Resource Type B",
				Type:        core.ScalarTypeBool,
				Description: "This is a test boolean annotation for resource type A that targets resource type B.",
				AppliesTo:   provider.LinkAnnotationResourceA,
			},
			"test/resourceTypeA::test.int.annotation": {
				Name:        "test.int.annotation",
				Label:       "Test Integer Annotation",
				Type:        core.ScalarTypeInteger,
				Description: "This is a test integer annotation for resource type A.",
				AppliesTo:   provider.LinkAnnotationResourceA,
				ValidateFunc: func(key string, annotationValue *core.ScalarValue) []*core.Diagnostic {
					intVal := core.IntValueFromScalar(annotationValue)
					if intVal > 800000 {
						return []*core.Diagnostic{
							{
								Level: core.DiagnosticLevelError,
								Message: fmt.Sprintf(
									"%s value exceeds maximum allowed value of 800000.",
									key,
								),
								Range: core.DiagnosticRangeFromSourceMeta(annotationValue.SourceMeta, nil),
							},
						}
					}
					return nil
				},
			},
			"test/resourceTypeB::test.bool.annotation": {
				Name:        "test.bool.annotation",
				Label:       "Test Boolean Annotation",
				Type:        core.ScalarTypeBool,
				Description: "This is a test boolean annotation for resource type B.",
				AppliesTo:   provider.LinkAnnotationResourceB,
			},
			"test/resourceTypeB::test.float.annotation": {
				Name:        "test.float.annotation",
				Label:       "Test Float Annotation",
				Type:        core.ScalarTypeFloat,
				Description: "This is a test float annotation for resource type B.",
				AppliesTo:   provider.LinkAnnotationResourceB,
			},
		},
	}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		// For test purposes only, does not reflect reality!
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testResourceTypeAResourceTypeBLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{}, nil
}

func (l *testResourceTypeAResourceTypeBLink) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	return &provider.LinkValidateOutput{}, nil
}

// testConfigurableLink is a mock link implementation with configurable
// return values for testing ValidateLinkConstraints.
type testConfigurableLink struct {
	linkType       string
	cardinality    *provider.LinkGetCardinalityOutput
	cardinalityErr error
	validateOutput *provider.LinkValidateOutput
	validateErr    error
	// capturedInput is populated by ValidateLink so tests can inspect
	// what was passed.
	capturedInput *provider.LinkValidateInput
}

func (l *testConfigurableLink) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return &provider.LinkStageChangesOutput{}, nil
}

func (l *testConfigurableLink) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource: provider.LinkPriorityResourceNone,
	}, nil
}

func (l *testConfigurableLink) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	return &provider.LinkGetTypeOutput{
		Type: l.linkType,
	}, nil
}

func (l *testConfigurableLink) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{}, nil
}

func (l *testConfigurableLink) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: map[string]*provider.LinkAnnotationDefinition{},
	}, nil
}

func (l *testConfigurableLink) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: provider.LinkKindHard,
	}, nil
}

func (l *testConfigurableLink) UpdateResourceA(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testConfigurableLink) UpdateResourceB(
	ctx context.Context,
	input *provider.LinkUpdateResourceInput,
) (*provider.LinkUpdateResourceOutput, error) {
	return &provider.LinkUpdateResourceOutput{}, nil
}

func (l *testConfigurableLink) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	return &provider.LinkUpdateIntermediaryResourcesOutput{}, nil
}

func (l *testConfigurableLink) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	return &provider.LinkGetIntermediaryExternalStateOutput{}, nil
}

func (l *testConfigurableLink) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	if l.cardinalityErr != nil {
		return nil, l.cardinalityErr
	}
	return l.cardinality, nil
}

func (l *testConfigurableLink) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	l.capturedInput = input
	if l.validateErr != nil {
		return nil, l.validateErr
	}
	if l.validateOutput != nil {
		return l.validateOutput, nil
	}
	return &provider.LinkValidateOutput{}, nil
}
