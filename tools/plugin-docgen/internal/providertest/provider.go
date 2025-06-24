package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func NewProvider() provider.Provider {
	return &providerv1.ProviderPluginDefinition{
		ProviderNamespace:        "test",
		ProviderConfigDefinition: providerConfigDefinition(),
		Resources: map[string]provider.Resource{
			"test/lambda/function": lambdaFunctionResource(),
			"test/dynamodb/table":  dynamodbTableResource(),
		},
		DataSources: map[string]provider.DataSource{
			"test/lambda/function": lambdaFunctionDataSource(),
		},
		Links: map[string]provider.Link{
			"test/lambda/function::test2/dynamodb/table": lambdaFunctionTest2DynamoDBTableLink(),
		},
		CustomVariableTypes: map[string]provider.CustomVariableType{
			"test/ec2/instanceType": ec2InstanceTypeVariable(),
		},
		Functions: map[string]provider.Function{
			"and":                         andFunction(),
			"compose":                     composeFunction(),
			"filter":                      filterFunction(),
			"create_specific_object_type": createSpecificObjectTypeFunction(),
			"produce_variadic_func":       produceVariadicFuncFunction(),
			"stringify":                   stringifyFunction(),
		},
	}
}

func providerConfigDefinition() *core.ConfigDefinition {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{
			"accessKeyId": {
				Type:        core.ScalarTypeString,
				Label:       "Access Key ID",
				Description: "The access key Id to use to authenticate with AWS.",
				Required:    true,
				Examples: []*core.ScalarValue{
					core.ScalarFromString("AKIAIOSFODNN7EXAMPLE"),
				},
			},
			"secretAccessKey": {
				Type:        core.ScalarTypeString,
				Label:       "Secret Access Key",
				Description: "The secret access key to use to authenticate with AWS.",
				Required:    true,
				Secret:      true,
			},
			"someConfigField": {
				Type:        core.ScalarTypeString,
				Label:       "Some Config Field",
				Description: "Some config field description.",
				Required:    false,
				AllowedValues: []*core.ScalarValue{
					core.ScalarFromString("Some value"),
					core.ScalarFromString("Another value"),
				},
				DefaultValue: core.ScalarFromString("Some value"),
			},
		},
		AllowAdditionalFields: true,
	}
}
