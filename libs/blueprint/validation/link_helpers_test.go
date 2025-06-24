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
				Required: true,
			},
			"test/resourceTypeA::test.string.<resourceTypeBName>.annotation": {
				Name:        "test.string.<resourceTypeBName>.annotation",
				Label:       "Test String Annotation for Resource Type B",
				Type:        core.ScalarTypeString,
				Description: "This is a test string annotation for resource type A that targets resource type B.",
			},
			"test/resourceTypeA::test.int.annotation": {
				Name:        "test.int.annotation",
				Label:       "Test Integer Annotation",
				Type:        core.ScalarTypeInteger,
				Description: "This is a test integer annotation for resource type A.",
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
			},
			"test/resourceTypeB::test.float.annotation": {
				Name:        "test.float.annotation",
				Label:       "Test Float Annotation",
				Type:        core.ScalarTypeFloat,
				Description: "This is a test float annotation for resource type B.",
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
