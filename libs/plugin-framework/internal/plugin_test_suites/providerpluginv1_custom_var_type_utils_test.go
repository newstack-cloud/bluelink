package plugintestsuites

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

func customVarTypeGetTypeInput() *provider.CustomVariableTypeGetTypeInput {
	return &provider.CustomVariableTypeGetTypeInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func customVarTypeGetDescriptionInput() *provider.CustomVariableTypeGetDescriptionInput {
	return &provider.CustomVariableTypeGetDescriptionInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func customVarTypeOptionsInput() *provider.CustomVariableTypeOptionsInput {
	return &provider.CustomVariableTypeOptionsInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}

func customVarTypeExamplesInput() *provider.CustomVariableTypeGetExamplesInput {
	return &provider.CustomVariableTypeGetExamplesInput{
		ProviderContext: testutils.CreateTestProviderContext("aws"),
	}
}
