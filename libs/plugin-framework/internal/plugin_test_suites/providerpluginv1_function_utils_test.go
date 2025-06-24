package plugintestsuites

import (
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

func functionGetDefinitionInput() *provider.FunctionGetDefinitionInput {
	return &provider.FunctionGetDefinitionInput{
		Params: testutils.CreateEmptyTestParams(),
	}
}
