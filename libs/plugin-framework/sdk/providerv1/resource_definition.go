package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
)

// ResourceDefinition is a template to be used for defining resources
// when creating provider plugins.
// It provides a structure that allows you to define a schema and behaviour
// of a resource.
// This implements the `provider.Resource` interface and can be used in the same way
// as any other resource implementation used in a provider plugin.
type ResourceDefinition struct {
	// The type of the resource, prefixed by the namespace of the provider.
	// Example: "aws/lambda/function" for the "aws" provider.
	Type string

	// A human-readable label for the resource type.
	// This will be used in documentation and tooling.
	Label string

	// A summary of the resource type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextSummary string

	// A summary of the resource type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedSummary string

	// A description of the resource type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextDescription string

	// A description of the resource type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedDescription string

	// A list of plain text examples that can be used to
	// demonstrate how to use the resource.
	// This will be used in documentation and tooling.
	PlainTextExamples []string

	// A list of markdown examples that can be used to
	// demonstrate how to use the resource.
	// This will be used in documentation and tooling.
	FormattedExamples []string

	// The schema of the resource specification that comes under the `spec` field
	// of a resource in a blueprint.
	Schema *provider.ResourceDefinitionsSchema

	// Holds the name of the field in the resource schema that holds the third-party
	// ID of the resource.
	// This is used to resolve references to a resource in a blueprint
	// where only the name of the resource is specified.
	// For example, references such as `resources.processOrderFunction` or
	// `processOrderFunction` should resolve to the ID of the resource in the blueprint.
	// The ID field must be a top-level property of the resource spec schema.
	IDField string

	// Specifies whether this resource is expected to have a common use-case
	// as a terminal resource that does not link out to other resources.
	// This is useful for providing useful warnings to users about their blueprints
	// without overloading them with warnings for all resources that don't have any outbound
	// links that could have.
	// If CommonTerminalFunc is provided, this static value will not be used.
	CommonTerminal bool

	// A function that can be used to dynamically determine whether a resource is likely
	// to be a common terminal resource (does not link out to other resources).
	CommonTerminalFunc func(
		ctx context.Context,
		input *provider.ResourceIsCommonTerminalInput,
	) (*provider.ResourceIsCommonTerminalOutput, error)

	// A static list of resource types that this resource can link to.
	// If ResourceCanLinkToFunc is provided, this static list will not be used.
	ResourceCanLinkTo []string

	// A function that can be used to dynamically determine what resource types this resource
	// can link to.
	ResourceCanLinkToFunc func(
		ctx context.Context,
		input *provider.ResourceCanLinkToInput,
	) (*provider.ResourceCanLinkToOutput, error)

	// A static list of resource types that must be stabilised
	// before this resource can be deployed when they are dependencies
	// of this resource.
	// If StabilisedDependenciesFunc is provided, this static list will not be used.
	StabilisedDependencies []string

	// A function that can be used to dynamically determine what resource types must be stabilised
	// before this resource can be deployed.
	StabilisedDependenciesFunc func(
		ctx context.Context,
		input *provider.ResourceStabilisedDependenciesInput,
	) (*provider.ResourceStabilisedDependenciesOutput, error)

	// A function to retrieve the current state of the resource from the upstream system.
	// (e.g. fetch the state of an Lambda Function from AWS)
	GetExternalStateFunc func(
		ctx context.Context,
		input *provider.ResourceGetExternalStateInput,
	) (*provider.ResourceGetExternalStateOutput, error)

	// A function to create the resource in the upstream provider.
	CreateFunc func(
		ctx context.Context,
		input *provider.ResourceDeployInput,
	) (*provider.ResourceDeployOutput, error)

	// A function to update the resource in the upstream provider.
	UpdateFunc func(
		ctx context.Context,
		input *provider.ResourceDeployInput,
	) (*provider.ResourceDeployOutput, error)

	// A function to delete the resource in the upstream provider.
	DestroyFunc func(
		ctx context.Context,
		input *provider.ResourceDestroyInput,
	) error

	// A function to apply custom validation for a resource
	// that goes beyond validating against the resource spec schema which
	// the blueprint framework takes care of.
	// When not provided, the plugin will return a result with an empty list
	// of diagnostics.
	CustomValidateFunc func(
		ctx context.Context,
		input *provider.ResourceValidateInput,
	) (*provider.ResourceValidateOutput, error)

	// A function that is used to determine whether or not a resource is considered
	// to be in a stable state.
	// When not provided, `true` will be returned for the resource type
	// whenever the deploy engine checks whether or not a resource has
	// stabilised.
	StabilisedFunc func(
		ctx context.Context,
		input *provider.ResourceHasStabilisedInput,
	) (*provider.ResourceHasStabilisedOutput, error)
}

func (r *ResourceDefinition) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	if r.CustomValidateFunc == nil {
		return &provider.ResourceValidateOutput{
			Diagnostics: []*core.Diagnostic{},
		}, nil
	}

	return r.CustomValidateFunc(ctx, input)
}

func (r *ResourceDefinition) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: &provider.ResourceSpecDefinition{
			Schema:  r.Schema,
			IDField: r.IDField,
		},
	}, nil
}

func (r *ResourceDefinition) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	if r.ResourceCanLinkToFunc != nil {
		return r.ResourceCanLinkToFunc(ctx, input)
	}

	return &provider.ResourceCanLinkToOutput{
		CanLinkTo: r.ResourceCanLinkTo,
	}, nil
}

func (r *ResourceDefinition) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	if r.StabilisedDependenciesFunc != nil {
		return r.StabilisedDependenciesFunc(ctx, input)
	}

	return &provider.ResourceStabilisedDependenciesOutput{
		StabilisedDependencies: r.StabilisedDependencies,
	}, nil
}

func (r *ResourceDefinition) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	if r.CommonTerminalFunc != nil {
		return r.CommonTerminalFunc(ctx, input)
	}

	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: r.CommonTerminal,
	}, nil
}

func (r *ResourceDefinition) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	return &provider.ResourceGetTypeOutput{
		Type:  r.Type,
		Label: r.Label,
	}, nil
}

func (r *ResourceDefinition) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	return &provider.ResourceGetTypeDescriptionOutput{
		PlainTextDescription: r.PlainTextDescription,
		MarkdownDescription:  r.FormattedDescription,
		PlainTextSummary:     r.PlainTextSummary,
		MarkdownSummary:      r.FormattedSummary,
	}, nil
}

func (r *ResourceDefinition) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  r.FormattedExamples,
		PlainTextExamples: r.PlainTextExamples,
	}, nil
}

func (r *ResourceDefinition) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	if r.CreateFunc == nil {
		return nil, errResourceCreateFunctionMissing(r.Type)
	}

	if r.UpdateFunc == nil {
		return nil, errResourceUpdateFunctionMissing(r.Type)
	}

	// The blueprint framework will only populate the `CurrentResourceState` field
	// of the input if the resource already exists in the blueprint state container,
	// meaning the resource is being updated.
	hasCurrentResourceState := isCurrentResourceStatePopulated(input)

	// If the changes provided require the resource to be re-created,
	// then we create the replacement resource and destroy the old one.
	// This does mean that for resources that have user-defined unique identifiers,
	// the user will need to ensure a new unique identifier is provided
	// in the resource spec.
	if hasCurrentResourceState && input.Changes.MustRecreate {
		resourceDeployOutput, err := r.CreateFunc(ctx, input)
		if err != nil {
			return nil, err
		}

		err = r.Destroy(ctx, &provider.ResourceDestroyInput{
			InstanceID:      input.InstanceID,
			ResourceID:      input.ResourceID,
			ResourceState:   input.Changes.AppliedResourceInfo.CurrentResourceState,
			ProviderContext: input.ProviderContext,
		})
		if err != nil {
			return nil, err
		}

		return resourceDeployOutput, nil
	}

	if hasCurrentResourceState {
		return r.UpdateFunc(ctx, input)
	}

	return r.CreateFunc(ctx, input)
}

func (r *ResourceDefinition) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	if r.StabilisedFunc != nil {
		return r.StabilisedFunc(ctx, input)
	}

	return &provider.ResourceHasStabilisedOutput{
		Stabilised: true,
	}, nil
}

func (r *ResourceDefinition) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	if r.GetExternalStateFunc == nil {
		return nil, errResourceGetExternalStateFunctionMissing(r.Type)
	}

	return r.GetExternalStateFunc(ctx, input)
}

func (r *ResourceDefinition) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	if r.DestroyFunc == nil {
		return errResourceDestroyFunctionMissing(r.Type)
	}

	return r.DestroyFunc(ctx, input)
}

func isCurrentResourceStatePopulated(
	input *provider.ResourceDeployInput,
) bool {
	return input != nil &&
		input.Changes != nil &&
		input.Changes.AppliedResourceInfo.CurrentResourceState != nil
}
