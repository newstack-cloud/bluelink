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
