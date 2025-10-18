package container

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/resourcehelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/state"
	"github.com/newstack-cloud/bluelink/libs/blueprint/subengine"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/common/testhelpers"
	"github.com/stretchr/testify/suite"
)

type ExpandResourceTemplatesTestSuite struct {
	specFixtureContainers          map[string]BlueprintContainer
	stateContainer                 state.Container
	funcRegistry                   provider.FunctionRegistry
	resourceRegistry               resourcehelpers.Registry
	dataSourceRegistry             provider.DataSourceRegistry
	providers                      map[string]provider.Provider
	resourceCache                  *core.Cache[*provider.ResolvedResource]
	resourceTemplateInputElemCache *core.Cache[[]*core.MappingNode]
	childExportFieldCache          *core.Cache[*subengine.ChildExportFieldInfo]
	suite.Suite
}

const (
	expandedOneToManyLinkFixtureName  = "expanded-one-to-many-link"
	expandedManyToOneLinkFixtureName  = "expanded-many-to-one-link"
	expandedManyToManyLinkFixtureName = "expanded-many-to-many-link"
	expandedFailureFixtureName        = "expanded-failure"
)

func (s *ExpandResourceTemplatesTestSuite) SetupSuite() {
	inputFiles := map[string]string{
		expandedOneToManyLinkFixtureName:  "__testdata/template-expansion/expanded-1-blueprint.yml",
		expandedManyToOneLinkFixtureName:  "__testdata/template-expansion/expanded-2-blueprint.yml",
		expandedManyToManyLinkFixtureName: "__testdata/template-expansion/expanded-3-blueprint.yml",
		expandedFailureFixtureName:        "__testdata/template-expansion/expanded-fail-blueprint.yml",
	}
	s.specFixtureContainers = make(map[string]BlueprintContainer)

	s.stateContainer = memstate.NewMemoryStateContainer()
	s.providers = map[string]provider.Provider{
		"aws": newTestAWSProvider(
			/* alwaysStabilise */ false,
			/* skipRetryFailuresForLinkNames */ []string{},
			s.stateContainer,
		),
		"core": providerhelpers.NewCoreProvider(
			s.stateContainer.Links(),
			core.BlueprintInstanceIDFromContext,
			os.Getwd,
			core.SystemClock{},
		),
	}
	logger, err := internal.NewTestLogger()
	if err != nil {
		s.FailNow(err.Error())
	}

	loader := NewDefaultLoader(
		s.providers,
		map[string]transform.SpecTransformer{},
		s.stateContainer,
		newFSChildResolver(),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(logger),
	)
	for name, filePath := range inputFiles {
		specBytes, err := os.ReadFile(filePath)
		if err != nil {
			s.FailNow(err.Error())
		}
		blueprintStr := string(specBytes)
		params := expandResourceTemplatesTestParams()
		bpContainer, err := loader.LoadString(context.TODO(), blueprintStr, schema.YAMLSpecFormat, params)
		if err != nil {
			s.FailNow(err.Error())
		}
		s.specFixtureContainers[name] = bpContainer
	}
}

func (s *ExpandResourceTemplatesTestSuite) SetupTest() {
	s.stateContainer = memstate.NewMemoryStateContainer()
	s.funcRegistry = provider.NewFunctionRegistry(s.providers)
	s.resourceRegistry = resourcehelpers.NewRegistry(
		s.providers,
		map[string]transform.SpecTransformer{},
		10*time.Millisecond,
		s.stateContainer,
		/* params */ nil,
	)
	s.dataSourceRegistry = provider.NewDataSourceRegistry(
		s.providers,
		core.SystemClock{},
		core.NewNopLogger(),
	)
	s.resourceCache = core.NewCache[*provider.ResolvedResource]()
	s.resourceTemplateInputElemCache = core.NewCache[[]*core.MappingNode]()
	s.childExportFieldCache = core.NewCache[*subengine.ChildExportFieldInfo]()
}

func (s *ExpandResourceTemplatesTestSuite) Test_expands_resource_template_with_one_to_many_link_relationship() {
	container := s.specFixtureContainers[expandedOneToManyLinkFixtureName]
	params := expandResourceTemplatesTestParams()
	subResolver := subengine.NewDefaultSubstitutionResolver(
		&subengine.Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		container.BlueprintSpec(),
		params,
	)

	ctx := context.TODO()
	linkChains, err := container.SpecLinkInfo().Links(ctx)
	s.Require().NoError(err)

	result, err := ExpandResourceTemplates(
		ctx,
		container.BlueprintSpec().Schema(),
		subResolver,
		linkChains,
		s.resourceTemplateInputElemCache,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *ExpandResourceTemplatesTestSuite) Test_expands_resource_template_with_many_to_one_link_relationship() {
	container := s.specFixtureContainers[expandedManyToOneLinkFixtureName]
	params := expandResourceTemplatesTestParams()
	subResolver := subengine.NewDefaultSubstitutionResolver(
		&subengine.Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		container.BlueprintSpec(),
		params,
	)

	ctx := context.TODO()
	linkChains, err := container.SpecLinkInfo().Links(ctx)
	s.Require().NoError(err)

	result, err := ExpandResourceTemplates(
		ctx,
		container.BlueprintSpec().Schema(),
		subResolver,
		linkChains,
		s.resourceTemplateInputElemCache,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *ExpandResourceTemplatesTestSuite) Test_expands_resource_template_with_many_to_many_link_relationship() {
	container := s.specFixtureContainers[expandedManyToManyLinkFixtureName]
	params := expandResourceTemplatesTestParams()
	subResolver := subengine.NewDefaultSubstitutionResolver(
		&subengine.Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		container.BlueprintSpec(),
		params,
	)

	ctx := context.TODO()
	linkChains, err := container.SpecLinkInfo().Links(ctx)
	s.Require().NoError(err)

	result, err := ExpandResourceTemplates(
		ctx,
		container.BlueprintSpec().Schema(),
		subResolver,
		linkChains,
		s.resourceTemplateInputElemCache,
	)
	s.Require().NoError(err)
	s.Require().NotNil(result)

	err = testhelpers.Snapshot(result)
	s.Require().NoError(err)
}

func (s *ExpandResourceTemplatesTestSuite) Test_fails_to_expand_when_link_relationship_between_templates_has_length_mismatch() {
	container := s.specFixtureContainers[expandedFailureFixtureName]
	params := expandResourceTemplatesTestParams()
	subResolver := subengine.NewDefaultSubstitutionResolver(
		&subengine.Registries{
			FuncRegistry:       s.funcRegistry,
			ResourceRegistry:   s.resourceRegistry,
			DataSourceRegistry: s.dataSourceRegistry,
		},
		s.stateContainer,
		provider.NewFileSourceRegistry(),
		s.resourceCache,
		s.resourceTemplateInputElemCache,
		s.childExportFieldCache,
		container.BlueprintSpec(),
		params,
	)

	ctx := context.TODO()
	linkChains, err := container.SpecLinkInfo().Links(ctx)
	s.Require().NoError(err)

	_, err = ExpandResourceTemplates(
		ctx,
		container.BlueprintSpec().Schema(),
		subResolver,
		linkChains,
		s.resourceTemplateInputElemCache,
	)
	s.Require().Error(err)
	runError, isRunError := err.(*errors.RunError)
	s.Assert().True(isRunError)
	s.Assert().Equal(ErrorReasonCodeResourceTemplateLinkLengthMismatch, runError.ReasonCode)
	s.Assert().Equal(
		"run error: resource template function has a link "+
			"to resource template ordersTable with a different input length,"+
			" links between resource templates can only be made when the "+
			"resolved items list from the `each` property of both templates"+
			" is of the same length",
		runError.Error(),
	)
}

func expandResourceTemplatesTestParams() core.BlueprintParams {
	environment := "production-env"
	tablesConfig := "[{\"name\":\"orders-1\"},{\"name\":\"orders-2\"},{\"name\":\"orders-3\"}]"
	functionsConfig := "[{\"handler\":\"ordersFunction-1\"},{\"handler\":\"ordersFunction-2\"},{\"handler\":\"ordersFunction-3\"}]"
	otherFunctionsConfig := "[{\"handler\":\"otherFunction-1\"}]"
	blueprintVars := map[string]*core.ScalarValue{
		"environment": {
			StringValue: &environment,
		},
		"tablesConfig": {
			StringValue: &tablesConfig,
		},
		"functionsConfig": {
			StringValue: &functionsConfig,
		},
		"otherFunctionsConfig": {
			StringValue: &otherFunctionsConfig,
		},
	}
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		blueprintVars,
	)
}

func TestExpandResourceTemplatesTestSuite(t *testing.T) {
	suite.Run(t, new(ExpandResourceTemplatesTestSuite))
}
