package convertv1

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sharedtypesv1"
)

// FromPBLinkCardinalityInfo converts a protobuf LinkCardinalityInfo response
// to a provider LinkGetCardinalityOutput.
func FromPBLinkCardinalityResponse(
	pbResponse *sharedtypesv1.LinkCardinalityResponse_CardinalityInfo,
) *provider.LinkGetCardinalityOutput {
	if pbResponse == nil || pbResponse.CardinalityInfo == nil {
		return nil
	}

	output := &provider.LinkGetCardinalityOutput{}

	if pbResponse.CardinalityInfo.CardinalityA != nil {
		output.CardinalityA = fromPBLinkCardinality(
			pbResponse.CardinalityInfo.CardinalityA,
		)
	}

	if pbResponse.CardinalityInfo.CardinalityB != nil {
		output.CardinalityB = fromPBLinkCardinality(
			pbResponse.CardinalityInfo.CardinalityB,
		)
	}

	return output
}

// FromPBLinkCardinalityResponseForAbstract converts a protobuf LinkCardinalityInfo
// response for an abstract link to a transformer AbstractLinkGetCardinalityOutput.
func FromPBLinkCardinalityResponseForAbstract(
	pbResponse *sharedtypesv1.LinkCardinalityResponse_CardinalityInfo,
) *transform.AbstractLinkGetCardinalityOutput {
	if pbResponse == nil || pbResponse.CardinalityInfo == nil {
		return nil
	}

	output := &transform.AbstractLinkGetCardinalityOutput{}

	if pbResponse.CardinalityInfo.CardinalityA != nil {
		output.CardinalityA = fromPBLinkCardinality(
			pbResponse.CardinalityInfo.CardinalityA,
		)
	}

	if pbResponse.CardinalityInfo.CardinalityB != nil {
		output.CardinalityB = fromPBLinkCardinality(
			pbResponse.CardinalityInfo.CardinalityB,
		)
	}

	return output
}

func fromPBLinkCardinality(
	pbCardinality *sharedtypesv1.LinkItemCardinality,
) provider.LinkCardinality {
	if pbCardinality == nil {
		return provider.LinkCardinality{}
	}

	return provider.LinkCardinality{
		Min: int(pbCardinality.Min),
		Max: int(pbCardinality.Max),
	}
}

// FromPBLinkCapabilitiesResponse converts a protobuf LinkCapabilitiesInfo response
// to a provider LinkGetCapabilitiesOutput.
func FromPBLinkCapabilitiesResponse(
	pbResponse *sharedtypesv1.LinkCapabilitiesResponse_CapabilitiesInfo,
) *provider.LinkGetCapabilitiesOutput {
	if pbResponse == nil || pbResponse.CapabilitiesInfo == nil {
		return &provider.LinkGetCapabilitiesOutput{}
	}

	return &provider.LinkGetCapabilitiesOutput{
		Provides: fromPBLinkCapabilities(pbResponse.CapabilitiesInfo.Provides),
		Requires: fromPBLinkCapabilities(pbResponse.CapabilitiesInfo.Requires),
	}
}

// ToPBLinkCapabilities converts a list of provider link capabilities to their
// protobuf representation.
func ToPBLinkCapabilities(
	capabilities []provider.LinkCapability,
) []*sharedtypesv1.LinkCapability {
	if len(capabilities) == 0 {
		return nil
	}

	pbCapabilities := make([]*sharedtypesv1.LinkCapability, 0, len(capabilities))
	for _, capability := range capabilities {
		pbCapabilities = append(pbCapabilities, &sharedtypesv1.LinkCapability{
			Name:      capability.Name,
			Resource:  int32(capability.Resource),
			MustExist: capability.MustExist,
		})
	}

	return pbCapabilities
}

func fromPBLinkCapabilities(
	pbCapabilities []*sharedtypesv1.LinkCapability,
) []provider.LinkCapability {
	if len(pbCapabilities) == 0 {
		return nil
	}

	capabilities := make([]provider.LinkCapability, 0, len(pbCapabilities))
	for _, pbCapability := range pbCapabilities {
		if pbCapability == nil {
			continue
		}
		capabilities = append(capabilities, provider.LinkCapability{
			Name:      pbCapability.Name,
			Resource:  provider.LinkPriorityResource(pbCapability.Resource),
			MustExist: pbCapability.MustExist,
		})
	}

	return capabilities
}
