package providerv1

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/utils"
)

// LinkDefinition is a template to be used for defining links
// between resources when creating provider plugins.
// It provides a structure that allows you to define a schema and behaviour
// of a link.
// This implements the `provider.Link` interface and can be used in the same way
// as any other link implementation used in a provider plugin.
type LinkDefinition struct {
	// The type of the first resource in the link relationship.
	// (e.g. "aws/lambda/function").
	ResourceTypeA string

	// The type of resource B in the link relationship.
	// (e.g. "aws/dynamodb/table").
	ResourceTypeB string

	// The kind of link that contributes to the ordering of resources during deployment.
	// This can either be "hard" or "soft".
	// Hard links require the priority resource must exist before the dependent resource
	// can be created.
	// Soft links do not require either of the resources to exist before the other.
	Kind provider.LinkKind

	// A summary of the link type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextSummary string

	// A summary of the link type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedSummary string

	// A description of the link type that is not formatted that can be used
	// to render descriptions in contexts that formatting is not supported.
	// This will be used in documentation and tooling.
	PlainTextDescription string

	// A description of the link type that can be formatted using markdown.
	// This will be used in documentation and tooling.
	FormattedDescription string

	// A mapping of annotation names prefixed by resource type that
	// can be used to fine tune the behaviour of a link in a blueprint spec.
	// The format should be as follows:
	// {resourceType}::{annotationName} -> LinkAnnotationDefinition
	// e.g. "aws/lambda/function::aws.lambda.dynamodb.accessType" -> LinkAnnotationDefinition
	AnnotationDefinitions map[string]*provider.LinkAnnotationDefinition

	// Cardinality on the A side of the link: how many B's each A may link to.
	// Zero value means no constraint.
	CardinalityA provider.LinkCardinality

	// Cardinality on the B side of the link: how many A's may link to each B.
	// Zero value means no constraint.
	CardinalityB provider.LinkCardinality

	// Provides lists the guarantees this link establishes about the resources in
	// its relationship once it has been deployed. A link that requires the same
	// capability on the same resource is deployed after this one, and destroyed
	// before it.
	//
	// Most links establish nothing that another link depends on and leave this
	// empty.
	Provides []provider.LinkCapability

	// Requires lists the guarantees this link needs established before it runs,
	// for example a Lambda function's VPC attachment being in place before the
	// link reads it to decide whether an endpoint is needed.
	//
	// A requirement that nothing in the blueprint provides is satisfied by
	// absence, so it is safe to declare unconditionally. Set MustExist on a
	// capability only where its absence makes the link itself invalid.
	Requires []provider.LinkCapability

	// Modifies declares which of the two resources in the relationship this link
	// writes to. The deployer takes an exclusive lock only on a declared side, and
	// the link scheduler only holds a link back while another link is busy with a
	// side it writes.
	//
	// Declaring this is what stops a resource that many links merely read from
	// serialising all of them: a VPC that forty placement links read, or a table
	// that a dozen access links read while writing only their own function.
	//
	// The zero value is LinkModifiesBoth, which is what the engine assumed before
	// links could declare this. Over-declaring costs throughput; under-declaring
	// lets two links write one resource at the same time.
	Modifies provider.LinkModifies

	// ValidateFunc is a custom validation function that runs at blueprint
	// validation time (pre-deploy). It receives the resource specs and
	// annotations for the link pair and returns diagnostics.
	// This is distinct from StageChangesFunc which runs at deployment
	// staging time. Nil means no custom validation.
	ValidateFunc func(
		ctx context.Context,
		input *provider.LinkValidateInput,
	) (*provider.LinkValidateOutput, error)

	// The priority resource in the relationship based on the ordering of the resource
	// types.
	// For example in the link type "aws/lambda/function::aws/dynamodb/table",
	// if the priority resource should be "aws/lambda/function", then this field
	// should be set to `provider.LinkPriorityResourceA`.
	// If there is no priority resource, this field should be set to
	// `provider.LinkPriorityResourceNone`.
	// This will not be used if PriorityResourceFunc is provided.
	PriorityResource provider.LinkPriorityResource

	// A function that can be used to dynamically determine the priority resource
	// in the link relationship.
	PriorityResourceFunc func(
		ctx context.Context,
		input *provider.LinkGetPriorityResourceInput,
	) (*provider.LinkGetPriorityResourceOutput, error)

	// A function that details the changes that will be made when a deployment of the loaded blueprint
	// for the link between two resources.
	// Unlike resources, links do not map to a specification for a single deployable unit,
	// so link implementations must specify the changes that will be made across multiple resources.
	StageChangesFunc func(
		ctx context.Context,
		input *provider.LinkStageChangesInput,
	) (*provider.LinkStageChangesOutput, error)

	// A function that returns what the link needs the specs of the blueprint-declared
	// resources it writes to say, once both of its endpoints have deployed.
	//
	// The framework merges the contributions every link makes to a resource and applies
	// them as a single update of that resource, so this performs no write of its own.
	// A link implements this or UpdateLinkedResourcesFunc for a given resource, never
	// both.
	//
	// Optional. A link that contributes to no resource leaves this unset and behaves
	// exactly as it did before contributions existed.
	ProduceResourceContributionsFunc func(
		ctx context.Context,
		input *provider.LinkProduceResourceContributionsInput,
	) (*provider.LinkProduceResourceContributionsOutput, error)

	// A function that deals with applying the changes to the blueprint-declared resources
	// in a link relationship, for the creation, update or removal of the link.
	// This is for changes that no contribution expresses.
	// The value of the `LinkData` field returned in the output will be combined
	// with the LinkData output from updating intermediary resources
	// to form the final LinkData that will be persisted in the state of the blueprint instance.
	// Parameters are passed in for extra context, blueprint variables will have already
	// been substituted at this stage and must be used instead of the passed in params argument
	// to ensure consistency between the staged changes that are reviewed and the deployment itself.
	UpdateLinkedResourcesFunc func(
		ctx context.Context,
		input *provider.LinkUpdateLinkedResourcesInput,
	) (*provider.LinkUpdateLinkedResourcesOutput, error)

	// A function that deals with creating, updating or deleting intermediary resources
	// that are required for the link between two resources.
	// This is called for both the creation and removal of a link between two resources.
	// The value of the `LinkData` field returned in the output will be combined
	// with the LinkData output from updating resource A and B
	// to form the final LinkData that will be persisted in the state of the blueprint instance.
	// Parameters are passed into UpdateIntermediaryResources for extra context, blueprint variables will have already
	// been substituted at this stage and must be used instead of the passed in params argument
	// to ensure consistency between the staged changes that are reviewed and the deployment itself.
	UpdateIntermediaryResourcesFunc func(
		ctx context.Context,
		input *provider.LinkUpdateIntermediaryResourcesInput,
	) (*provider.LinkUpdateIntermediaryResourcesOutput, error)

	// A function that fetches the current cloud state for intermediary
	// resources owned by this link. Used for drift detection and reconciliation.
	// Link implementations that don't manage intermediary resources should leave
	// this nil, which will return an empty result.
	GetIntermediaryExternalStateFunc func(
		ctx context.Context,
		input *provider.LinkGetIntermediaryExternalStateInput,
	) (*provider.LinkGetIntermediaryExternalStateOutput, error)
}

func (l *LinkDefinition) GetType(
	ctx context.Context,
	input *provider.LinkGetTypeInput,
) (*provider.LinkGetTypeOutput, error) {
	linkType := core.LinkType(l.ResourceTypeA, l.ResourceTypeB)
	return &provider.LinkGetTypeOutput{
		Type: linkType,
	}, nil
}

func (l *LinkDefinition) GetKind(
	ctx context.Context,
	input *provider.LinkGetKindInput,
) (*provider.LinkGetKindOutput, error) {
	return &provider.LinkGetKindOutput{
		Kind: l.Kind,
	}, nil
}

func (l *LinkDefinition) GetTypeDescription(
	ctx context.Context,
	input *provider.LinkGetTypeDescriptionInput,
) (*provider.LinkGetTypeDescriptionOutput, error) {
	return &provider.LinkGetTypeDescriptionOutput{
		PlainTextSummary:     l.PlainTextSummary,
		MarkdownSummary:      l.FormattedSummary,
		PlainTextDescription: l.PlainTextDescription,
		MarkdownDescription:  l.FormattedDescription,
	}, nil
}

func (l *LinkDefinition) GetAnnotationDefinitions(
	ctx context.Context,
	input *provider.LinkGetAnnotationDefinitionsInput,
) (*provider.LinkGetAnnotationDefinitionsOutput, error) {
	return &provider.LinkGetAnnotationDefinitionsOutput{
		AnnotationDefinitions: l.AnnotationDefinitions,
	}, nil
}

func (l *LinkDefinition) GetPriorityResource(
	ctx context.Context,
	input *provider.LinkGetPriorityResourceInput,
) (*provider.LinkGetPriorityResourceOutput, error) {
	if l.PriorityResourceFunc != nil {
		return l.PriorityResourceFunc(ctx, input)
	}

	priorityResourceType := l.getPriorityResourceType()
	return &provider.LinkGetPriorityResourceOutput{
		PriorityResource:     l.PriorityResource,
		PriorityResourceType: priorityResourceType,
	}, nil
}

func (l *LinkDefinition) StageChanges(
	ctx context.Context,
	input *provider.LinkStageChangesInput,
) (*provider.LinkStageChangesOutput, error) {
	return l.StageChangesFunc(ctx, input)
}

func (l *LinkDefinition) ProduceResourceContributions(
	ctx context.Context,
	input *provider.LinkProduceResourceContributionsInput,
) (*provider.LinkProduceResourceContributionsOutput, error) {
	if l.ProduceResourceContributionsFunc == nil {
		// Contributing nothing is the behaviour of every link that has not moved to
		// contributions, so it is an empty result rather than an error.
		return &provider.LinkProduceResourceContributionsOutput{}, nil
	}

	// Attach link ID to the context so that it can be automatically
	// attached to calls from the plugin to the plugin service, which matters here
	// because producing a contribution can create a resource the link owns and
	// acquire locks to do so.
	ctxWithLinkID := context.WithValue(ctx, utils.ContextKeyLinkID, input.LinkID)

	return l.ProduceResourceContributionsFunc(ctxWithLinkID, input)
}

func (l *LinkDefinition) UpdateLinkedResources(
	ctx context.Context,
	input *provider.LinkUpdateLinkedResourcesInput,
) (*provider.LinkUpdateLinkedResourcesOutput, error) {
	return l.UpdateLinkedResourcesFunc(ctx, input)
}

func (l *LinkDefinition) UpdateIntermediaryResources(
	ctx context.Context,
	input *provider.LinkUpdateIntermediaryResourcesInput,
) (*provider.LinkUpdateIntermediaryResourcesOutput, error) {
	// Attach link ID to the context so that it can be automatically
	// attached to calls from the plugin to the plugin service,
	// this is especially useful for ensuring the link ID is always attached
	// as the "acquiredBy" field when acquiring a resource lock.
	ctxWithLinkID := context.WithValue(ctx, utils.ContextKeyLinkID, input.LinkID)
	return l.UpdateIntermediaryResourcesFunc(ctxWithLinkID, input)
}

func (l *LinkDefinition) GetIntermediaryExternalState(
	ctx context.Context,
	input *provider.LinkGetIntermediaryExternalStateInput,
) (*provider.LinkGetIntermediaryExternalStateOutput, error) {
	if l.GetIntermediaryExternalStateFunc == nil {
		// Return empty result for links that don't manage intermediary resources
		return &provider.LinkGetIntermediaryExternalStateOutput{
			IntermediaryStates: nil,
		}, nil
	}
	return l.GetIntermediaryExternalStateFunc(ctx, input)
}

func (l *LinkDefinition) GetCardinality(
	ctx context.Context,
	input *provider.LinkGetCardinalityInput,
) (*provider.LinkGetCardinalityOutput, error) {
	return &provider.LinkGetCardinalityOutput{
		CardinalityA: l.CardinalityA,
		CardinalityB: l.CardinalityB,
	}, nil
}

func (l *LinkDefinition) GetCapabilities(
	ctx context.Context,
	input *provider.LinkGetCapabilitiesInput,
) (*provider.LinkGetCapabilitiesOutput, error) {
	return &provider.LinkGetCapabilitiesOutput{
		Provides: l.Provides,
		Requires: l.Requires,
		Modifies: l.Modifies,
	}, nil
}

func (l *LinkDefinition) ValidateLink(
	ctx context.Context,
	input *provider.LinkValidateInput,
) (*provider.LinkValidateOutput, error) {
	if l.ValidateFunc == nil {
		// No custom validation, return empty diagnostics
		return &provider.LinkValidateOutput{
			Diagnostics: nil,
		}, nil
	}
	return l.ValidateFunc(ctx, input)
}

func (l *LinkDefinition) getPriorityResourceType() string {
	switch l.PriorityResource {
	case provider.LinkPriorityResourceA:
		return l.ResourceTypeA
	case provider.LinkPriorityResourceB:
		return l.ResourceTypeB
	default:
		return ""
	}
}
