package container

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
)

// transformerResourceProvider adapts a spec transformer to the provider.Provider
// interface so the link info engine can resolve transformer abstract resource
// types (e.g. "celerity/handler") that do not belong to a provider namespace.
// Only the resource resolution surface is functional, everything else is
// out of scope for abstract resources.
type transformerResourceProvider struct {
	transformer transform.SpecTransformer
}

func newTransformerResourceProvider(
	transformer transform.SpecTransformer,
) provider.Provider {
	return &transformerResourceProvider{
		transformer: transformer,
	}
}

func (p *transformerResourceProvider) Namespace(ctx context.Context) (string, error) {
	return "", errTransformerProviderUnsupported("Namespace")
}

func (p *transformerResourceProvider) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return p.transformer.ConfigDefinition(ctx)
}

func (p *transformerResourceProvider) Resource(
	ctx context.Context,
	resourceType string,
) (provider.Resource, error) {
	abstractResource, err := p.transformer.AbstractResource(ctx, resourceType)
	if err != nil {
		return nil, err
	}
	if abstractResource == nil {
		return nil, fmt.Errorf(
			"abstract resource type %q was not found in the transformer",
			resourceType,
		)
	}
	return &abstractResourceAdapter{abstract: abstractResource}, nil
}

func (p *transformerResourceProvider) DataSource(
	ctx context.Context,
	dataSourceType string,
) (provider.DataSource, error) {
	return nil, errTransformerProviderUnsupported("DataSource")
}

func (p *transformerResourceProvider) Link(
	ctx context.Context,
	resourceTypeA string,
	resourceTypeB string,
) (provider.Link, error) {
	return nil, errTransformerProviderUnsupported("Link")
}

func (p *transformerResourceProvider) CustomVariableType(
	ctx context.Context,
	customVariableType string,
) (provider.CustomVariableType, error) {
	return nil, errTransformerProviderUnsupported("CustomVariableType")
}

func (p *transformerResourceProvider) Function(
	ctx context.Context,
	functionName string,
) (provider.Function, error) {
	return nil, errTransformerProviderUnsupported("Function")
}

func (p *transformerResourceProvider) ListResourceTypes(ctx context.Context) ([]string, error) {
	return p.transformer.ListAbstractResourceTypes(ctx)
}

func (p *transformerResourceProvider) ListLinkTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *transformerResourceProvider) ListDataSourceTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *transformerResourceProvider) ListCustomVariableTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *transformerResourceProvider) ListFunctions(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (p *transformerResourceProvider) RetryPolicy(ctx context.Context) (*provider.RetryPolicy, error) {
	return nil, nil
}

// abstractResourceAdapter exposes a transformer abstract resource through the
// provider.Resource interface for pre-transform surfaces such as link
// information (spec definitions, link metadata and type information).
// Deploy-time methods error as abstract resources must be transformed into
// concrete resources before staging changes or deploying.
type abstractResourceAdapter struct {
	abstract transform.AbstractResource
}

func (r *abstractResourceAdapter) CustomValidate(
	ctx context.Context,
	input *provider.ResourceValidateInput,
) (*provider.ResourceValidateOutput, error) {
	// Abstract resource custom validation runs through the resource registry's
	// transformer fallback with a transformer context, this adapter only has
	// a provider context available so it stays neutral.
	return &provider.ResourceValidateOutput{}, nil
}

func (r *abstractResourceAdapter) GetSpecDefinition(
	ctx context.Context,
	input *provider.ResourceGetSpecDefinitionInput,
) (*provider.ResourceGetSpecDefinitionOutput, error) {
	output, err := r.abstract.GetSpecDefinition(
		ctx,
		&transform.AbstractResourceGetSpecDefinitionInput{},
	)
	if err != nil {
		return nil, err
	}
	return &provider.ResourceGetSpecDefinitionOutput{
		SpecDefinition: output.SpecDefinition,
	}, nil
}

func (r *abstractResourceAdapter) CanLinkTo(
	ctx context.Context,
	input *provider.ResourceCanLinkToInput,
) (*provider.ResourceCanLinkToOutput, error) {
	output, err := r.abstract.CanLinkTo(
		ctx,
		&transform.AbstractResourceCanLinkToInput{},
	)
	if err != nil {
		return nil, err
	}
	return &provider.ResourceCanLinkToOutput{CanLinkTo: output.CanLinkTo}, nil
}

func (r *abstractResourceAdapter) GetStabilisedDependencies(
	ctx context.Context,
	input *provider.ResourceStabilisedDependenciesInput,
) (*provider.ResourceStabilisedDependenciesOutput, error) {
	return &provider.ResourceStabilisedDependenciesOutput{
		StabilisedDependencies: []string{},
	}, nil
}

func (r *abstractResourceAdapter) IsCommonTerminal(
	ctx context.Context,
	input *provider.ResourceIsCommonTerminalInput,
) (*provider.ResourceIsCommonTerminalOutput, error) {
	output, err := r.abstract.IsCommonTerminal(
		ctx,
		&transform.AbstractResourceIsCommonTerminalInput{},
	)
	if err != nil {
		return nil, err
	}
	return &provider.ResourceIsCommonTerminalOutput{
		IsCommonTerminal: output.IsCommonTerminal,
	}, nil
}

func (r *abstractResourceAdapter) GetType(
	ctx context.Context,
	input *provider.ResourceGetTypeInput,
) (*provider.ResourceGetTypeOutput, error) {
	output, err := r.abstract.GetType(ctx, &transform.AbstractResourceGetTypeInput{})
	if err != nil {
		return nil, err
	}
	return &provider.ResourceGetTypeOutput{
		Type:  output.Type,
		Label: output.Label,
	}, nil
}

func (r *abstractResourceAdapter) GetTypeDescription(
	ctx context.Context,
	input *provider.ResourceGetTypeDescriptionInput,
) (*provider.ResourceGetTypeDescriptionOutput, error) {
	output, err := r.abstract.GetTypeDescription(
		ctx,
		&transform.AbstractResourceGetTypeDescriptionInput{},
	)
	if err != nil {
		return nil, err
	}
	return &provider.ResourceGetTypeDescriptionOutput{
		MarkdownDescription:  output.MarkdownDescription,
		PlainTextDescription: output.PlainTextDescription,
		MarkdownSummary:      output.MarkdownSummary,
		PlainTextSummary:     output.PlainTextSummary,
	}, nil
}

func (r *abstractResourceAdapter) GetExamples(
	ctx context.Context,
	input *provider.ResourceGetExamplesInput,
) (*provider.ResourceGetExamplesOutput, error) {
	output, err := r.abstract.GetExamples(
		ctx,
		&transform.AbstractResourceGetExamplesInput{},
	)
	if err != nil {
		return nil, err
	}
	return &provider.ResourceGetExamplesOutput{
		MarkdownExamples:  output.MarkdownExamples,
		PlainTextExamples: output.PlainTextExamples,
	}, nil
}

func (r *abstractResourceAdapter) Deploy(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return nil, errAbstractResourceNotDeployable()
}

func (r *abstractResourceAdapter) HasStabilised(
	ctx context.Context,
	input *provider.ResourceHasStabilisedInput,
) (*provider.ResourceHasStabilisedOutput, error) {
	return nil, errAbstractResourceNotDeployable()
}

func (r *abstractResourceAdapter) GetExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return nil, errAbstractResourceNotDeployable()
}

func (r *abstractResourceAdapter) Destroy(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return errAbstractResourceNotDeployable()
}

func errTransformerProviderUnsupported(method string) error {
	return fmt.Errorf(
		"%s is not supported by a transformer-backed resource provider, "+
			"it only resolves abstract resource types for link information",
		method,
	)
}

func errAbstractResourceNotDeployable() error {
	return fmt.Errorf(
		"an abstract resource reached a deploy-time code path, " +
			"abstract resources must be transformed into concrete resources before staging or deployment",
	)
}
