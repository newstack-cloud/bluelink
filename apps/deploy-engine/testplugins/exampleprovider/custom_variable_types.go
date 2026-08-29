package main

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

// The regions are close to one another in name so that a value with a typo in it
// (e.g. "eu-central-4") has one clear suggestion rather than several.
var regionOptions = []string{
	"us-east-1",
	"us-east-2",
	"us-west-1",
	"us-west-2",
	"eu-west-1",
	"eu-west-2",
	"eu-central-1",
}

var instanceSizeOptions = []string{
	"small",
	"medium",
	"large",
	"xlarge",
}

var environmentOptions = []string{
	"development",
	"staging",
	"production",
}

func customVarTypeRegion() provider.CustomVariableType {
	return customVarType(
		"example/region",
		"Example Region",
		"A region to deploy the example service to.",
		regionOptions,
	)
}

func customVarTypeInstanceSize() provider.CustomVariableType {
	return customVarType(
		"example/instanceSize",
		"Example Instance Size",
		"The size of the instance to run the example service on.",
		instanceSizeOptions,
	)
}

func customVarTypeEnvironment() provider.CustomVariableType {
	return customVarType(
		"example/environment",
		"Example Environment",
		"The environment to deploy the example service to.",
		environmentOptions,
	)
}

func customVarType(
	varType string,
	label string,
	description string,
	options []string,
) provider.CustomVariableType {
	return &providerv1.CustomVariableTypeDefinition{
		Type:                 varType,
		Label:                label,
		PlainTextSummary:     description,
		FormattedSummary:     description,
		PlainTextDescription: description,
		FormattedDescription: description,
		CustomVarTypeOptions: customVarTypeOptions(options),
	}
}

func customVarTypeOptions(
	options []string,
) map[string]*provider.CustomVariableTypeOption {
	optionMap := map[string]*provider.CustomVariableTypeOption{}
	for _, option := range options {
		optionMap[option] = &provider.CustomVariableTypeOption{
			Label: option,
			Value: core.ScalarFromString(option),
		}
	}

	return optionMap
}
