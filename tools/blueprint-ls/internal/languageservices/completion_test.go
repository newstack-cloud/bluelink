package languageservices

import (
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/corefunctions"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

// CompletionServiceGetItemsSuite tests the completion service GetCompletionItems method.
type CompletionServiceGetItemsSuite struct {
	suite.Suite
	service *CompletionService
}

func (s *CompletionServiceGetItemsSuite) SetupTest() {
	logger, err := zap.NewDevelopment()
	if err != nil {
		s.FailNow(err.Error())
	}

	state := NewState()
	state.SetLinkSupportCapability(true)
	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/dynamodb/table": &testutils.DynamoDBTableResource{},
		},
	}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{
		DataSources: map[string]provider.DataSource{
			"aws/vpc": &testutils.VPCDataSource{},
		},
	}
	customVarTypeRegistry := &testutils.CustomVarTypeRegistryMock{
		CustomVarTypes: map[string]provider.CustomVariableType{
			"aws/ec2/instanceType": &testutils.InstanceTypeCustomVariableType{},
		},
	}
	functionRegistry := &testutils.FunctionRegistryMock{
		Functions: map[string]provider.Function{
			"len": corefunctions.NewLenFunction(),
		},
	}
	s.service = NewCompletionService(resourceRegistry, dataSourceRegistry, customVarTypeRegistry, functionRegistry, nil, state, logger)
}

func TestCompletionServiceGetItemsSuite(t *testing.T) {
	suite.Run(t, new(CompletionServiceGetItemsSuite))
}

