package providertest

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/sdk/providerv1"
)

func ec2InstanceTypeVariable() provider.CustomVariableType {
	return &providerv1.CustomVariableTypeDefinition{
		Type:                 "test/ec2/instanceType",
		Label:                "Amazon EC2 Instance Type",
		PlainTextSummary:     "An Amazon EC2 instance type for a VM.",
		FormattedDescription: "A custom variable type that represents an Amazon EC2 instance type for a virtual machine.",
		CustomVarTypeOptions: map[string]*provider.CustomVariableTypeOption{
			"t2.micro": {
				Value:       core.ScalarFromString("t2.micro"),
				Label:       "t2.micro",
				Description: "A t2.micro instance type.",
			},
			"t2.small": {
				Value:       core.ScalarFromString("t2.small"),
				Label:       "t2.small",
				Description: "A t2.small instance type.",
			},
			"t2.medium": {
				Value:       core.ScalarFromString("t2.medium"),
				Label:       "t2.medium",
				Description: "A t2.medium instance type.",
			},
			"t2.large": {
				Value:       core.ScalarFromString("t2.large"),
				Label:       "t2.large",
				Description: "A t2.large instance type.",
			},
		},
		FormattedExamples: []string{
			"```yaml\nvariables:\n  - name: instanceType\n    type: test/ec2/instanceType\n    value: t2.micro\n```",
			"```yaml\nvariables:\n  - name: instanceType\n    type: test/ec2/instanceType\n    value: t2.medium\n```",
		},
	}
}
