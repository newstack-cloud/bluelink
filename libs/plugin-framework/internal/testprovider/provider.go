package testprovider

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

// NewProvider creates a new instance of the test AWS provider
// that contains the supported resources, links, custom variable types
// and functions for the stub AWS provider.
// This is purely for testing purposes and does not interact with AWS services
// or provide functionality that would reflect that of a real AWS provider
// implementation.
func NewProvider() provider.Provider {
	lambdaFuncDDBTableLinkType := core.LinkType(
		"aws/lambda/function",
		"aws/dynamodb/table",
	)
	return &providerv1.ProviderPluginDefinition{
		ProviderNamespace:        "aws",
		ProviderConfigDefinition: TestProviderConfigDefinition(),
		Resources: map[string]provider.Resource{
			"aws/lambda/function": resourceLambdaFunction(),
		},
		DataSources: map[string]provider.DataSource{
			"aws/vpc": dataSourceVPC(),
		},
		Links: map[string]provider.Link{
			lambdaFuncDDBTableLinkType: linkLambdaFunctionDynamoDBTable(),
		},
		CustomVariableTypes: map[string]provider.CustomVariableType{
			"aws/ec2/instanceType": customVarTypeEC2InstanceType(),
		},
		Functions:           Functions(),
		ProviderRetryPolicy: TestProviderRetryPolicy(),
	}
}

// Functions returns a map of function names to their implementations
// for the test provider.
func Functions() map[string]provider.Function {
	return map[string]provider.Function{
		"trim_suffix":           functionTrimSuffix(),
		"call_self":             functionCallSelf(),
		"trim_space_and_suffix": functionTrimSpaceAndSuffix(),
		"alter_list":            functionAlterList(),
		"alter_map":             functionAlterMap(),
		"alter_object":          functionAlterObject(),
		"compose":               functionCompose(),
		"map":                   functionMap(),
	}
}

// TestProviderConfigDefinition creates the config definition for the test AWS provider.
func TestProviderConfigDefinition() *core.ConfigDefinition {
	return &core.ConfigDefinition{
		Fields: map[string]*core.ConfigFieldDefinition{
			"accessKeyId": {
				Type:        core.ScalarTypeString,
				Label:       "Access Key ID",
				Description: "The access key ID for the AWS account to connect to.",
				Examples: []*core.ScalarValue{
					core.ScalarFromString("AKIAEXAMPLEACCESSKEYID"),
				},
				Required: true,
			},
			"secretAccessKey": {
				Type:        core.ScalarTypeString,
				Label:       "Secret Access Key",
				Description: "The secret access key for the AWS account to connect to.",
				Required:    true,
			},
			"region": {
				Type:        core.ScalarTypeString,
				Label:       "Region",
				Description: "The AWS region to connect to.",
				Examples: []*core.ScalarValue{
					core.ScalarFromString("us-west-2"),
				},
				AllowedValues: awsRegions(),
				Required:      false,
			},
		},
	}
}

// TestProviderRetryPolicy creates the retry policy for the test AWS provider.
func TestProviderRetryPolicy() *provider.RetryPolicy {
	return &provider.RetryPolicy{
		MaxRetries:      3,
		FirstRetryDelay: 4,
		MaxDelay:        200,
		BackoffFactor:   1.5,
		Jitter:          true,
	}
}

func awsRegions() []*core.ScalarValue {
	return []*core.ScalarValue{
		core.ScalarFromString("us-east-1"),
		core.ScalarFromString("us-east-2"),
		core.ScalarFromString("us-west-1"),
		core.ScalarFromString("us-west-2"),
		core.ScalarFromString("ap-south-1"),
		core.ScalarFromString("ap-northeast-1"),
		core.ScalarFromString("ap-northeast-2"),
		core.ScalarFromString("ap-southeast-1"),
		core.ScalarFromString("ap-southeast-2"),
		core.ScalarFromString("ap-northeast-3"),
		core.ScalarFromString("ca-central-1"),
		core.ScalarFromString("eu-central-1"),
		core.ScalarFromString("eu-west-1"),
		core.ScalarFromString("eu-west-2"),
	}
}
