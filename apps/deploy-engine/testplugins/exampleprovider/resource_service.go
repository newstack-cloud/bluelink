package main

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

// A blueprint needs at least one resource to be worth validating, this resource
// exists so that a blueprint that exercises the custom variable types has
// something to deploy. It does not talk to an upstream system, deploying it
// reports success without doing anything.
func resourceService() provider.Resource {
	description := "An example service."

	return &providerv1.ResourceDefinition{
		Type:                 "example/service",
		Label:                "Example Service",
		PlainTextSummary:     description,
		FormattedSummary:     description,
		PlainTextDescription: description,
		FormattedDescription: description,
		IDField:              "id",
		Schema: &provider.ResourceDefinitionsSchema{
			Type: provider.ResourceDefinitionsSchemaTypeObject,
			Attributes: map[string]*provider.ResourceDefinitionsSchema{
				"serviceName": {
					Type:        provider.ResourceDefinitionsSchemaTypeString,
					Label:       "Service Name",
					Description: "The name of the example service.",
				},
				"region": {
					Type:        provider.ResourceDefinitionsSchemaTypeString,
					Label:       "Region",
					Description: "The region to deploy the example service to.",
				},
				"id": {
					Type:     provider.ResourceDefinitionsSchemaTypeString,
					Label:    "ID",
					Computed: true,
				},
			},
			Required: []string{"serviceName"},
		},
		GetExternalStateFunc: getServiceExternalState,
		CreateFunc:           deployService,
		UpdateFunc:           deployService,
		DestroyFunc:          destroyService,
	}
}

func getServiceExternalState(
	ctx context.Context,
	input *provider.ResourceGetExternalStateInput,
) (*provider.ResourceGetExternalStateOutput, error) {
	return &provider.ResourceGetExternalStateOutput{
		ResourceSpecState: &core.MappingNode{
			Fields: map[string]*core.MappingNode{},
		},
	}, nil
}

func deployService(
	ctx context.Context,
	input *provider.ResourceDeployInput,
) (*provider.ResourceDeployOutput, error) {
	return &provider.ResourceDeployOutput{
		ComputedFieldValues: map[string]*core.MappingNode{
			"spec.id": core.MappingNodeFromString("example-service-id"),
		},
	}, nil
}

func destroyService(
	ctx context.Context,
	input *provider.ResourceDestroyInput,
) error {
	return nil
}
