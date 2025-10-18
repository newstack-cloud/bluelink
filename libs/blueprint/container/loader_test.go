package container

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal"
	"github.com/newstack-cloud/bluelink/libs/blueprint/internal/memstate"
	"github.com/newstack-cloud/bluelink/libs/blueprint/links"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/providerhelpers"
	"github.com/newstack-cloud/bluelink/libs/blueprint/refgraph"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"github.com/stretchr/testify/suite"
)

type LoaderTestSuite struct {
	specFixtures                 map[string]string
	specFixtureFiles             map[string]string
	specFixtureSchemas           map[string]*schema.Blueprint
	loader                       Loader
	loaderDefaultCore            Loader
	loaderValidateAfterTransform Loader
	providersWithoutCore         map[string]provider.Provider
	specTransformers             map[string]transform.SpecTransformer
	logger                       core.Logger
	suite.Suite
}

const (
	validServerlessBlueprintName = "valid-serverless"
)

func (s *LoaderTestSuite) SetupSuite() {
	s.specFixtures = make(map[string]string)
	s.specFixtureFiles = map[string]string{
		"valid":                       "__testdata/loader/valid-blueprint.yml",
		"invalid-yaml":                "__testdata/loader/invalid-yaml-blueprint.yml",
		"invalid-schema":              "__testdata/loader/invalid-schema-blueprint.yml",
		"unsupported-var-type":        "__testdata/loader/unsupported-var-type-blueprint.yml",
		validServerlessBlueprintName:  "__testdata/loader/valid-serverless-blueprint.yml",
		"missing-transform":           "__testdata/loader/missing-transform-blueprint.yml",
		"cyclic-ref":                  "__testdata/loader/cyclic-ref-blueprint.yml",
		"cyclic-ref-2":                "__testdata/loader/cyclic-ref-2-blueprint.yml",
		"cyclic-ref-3":                "__testdata/loader/cyclic-ref-3-blueprint.yml",
		"cyclic-ref-4":                "__testdata/loader/cyclic-ref-4-blueprint.yml",
		"cyclic-ref-5":                "__testdata/loader/cyclic-ref-5-blueprint.yml",
		"cyclic-ref-6":                "__testdata/loader/cyclic-ref-6-blueprint.yml",
		"invalid-resource-each-dep-1": "__testdata/loader/invalid-resource-each-dep-1-blueprint.yml",
		"invalid-resource-each-dep-2": "__testdata/loader/invalid-resource-each-dep-2-blueprint.yml",
		"stub-resource":               "__testdata/loader/stub-resource-blueprint.yml",
	}
	s.specFixtureSchemas = make(map[string]*schema.Blueprint)

	for name, filePath := range s.specFixtureFiles {
		specBytes, err := os.ReadFile(filePath)
		if err != nil {
			s.FailNow(err.Error())
		}
		blueprintStr := string(specBytes)
		s.specFixtures[name] = blueprintStr
		if strings.HasPrefix(name, "valid") {
			blueprint, err := schema.LoadString(blueprintStr, schema.YAMLSpecFormat)
			if err != nil {
				s.FailNow(err.Error())
			}
			s.specFixtureSchemas[name] = blueprint
		}
	}

	stateContainer := memstate.NewMemoryStateContainer()
	providers := map[string]provider.Provider{
		"aws": newTestAWSProvider(
			/* alwaysStabilise */ false,
			/* skipRetryFailuresForLinkNames */ []string{},
			stateContainer,
		),
		"core": providerhelpers.NewCoreProvider(
			stateContainer.Links(),
			core.BlueprintInstanceIDFromContext,
			os.Getwd,
			provider.NewFileSourceRegistry(),
			core.SystemClock{},
		),
	}
	specTransformers := map[string]transform.SpecTransformer{
		"serverless-2024": &internal.ServerlessTransformer{},
	}
	s.specTransformers = specTransformers
	logger := core.NewNopLogger()
	s.logger = logger
	s.loader = NewDefaultLoader(
		providers,
		specTransformers,
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(logger),
	)
	s.loaderValidateAfterTransform = NewDefaultLoader(
		providers,
		specTransformers,
		stateContainer,
		newFSChildResolver(),
		WithLoaderTransformSpec(true),
		WithLoaderValidateAfterTransform(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(logger),
	)
	providersWithoutCore := map[string]provider.Provider{
		"aws": newTestAWSProvider(
			/* alwaysStabilise */ false,
			/* skipRetryFailuresForLinkNames */ []string{},
			stateContainer,
		),
	}
	s.providersWithoutCore = providersWithoutCore

	s.loaderDefaultCore = NewDefaultLoader(
		providersWithoutCore,
		specTransformers,
		stateContainer,
		newFSChildResolver(),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(logger),
	)
}

func (s *LoaderTestSuite) Test_loads_container_from_input_spec_file_without_any_issues() {
	container, err := s.loader.Load(context.TODO(), s.specFixtureFiles["valid"], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(container)
}

func (s *LoaderTestSuite) Test_loads_container_from_input_spec_file_using_default_core_provider() {
	container, err := s.loaderDefaultCore.Load(context.TODO(), s.specFixtureFiles["valid"], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(container)
}

func (s *LoaderTestSuite) Test_loads_container_from_input_spec_string_without_any_issues() {
	container, err := s.loader.LoadString(context.TODO(), s.specFixtures["valid"], schema.YAMLSpecFormat, createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(container)
}

func (s *LoaderTestSuite) Test_validates_spec_from_input_spec_file_without_any_issues() {
	validationRes, err := s.loader.Validate(context.TODO(), s.specFixtureFiles["valid"], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(validationRes)
}

func (s *LoaderTestSuite) Test_validates_spec_from_input_spec_string_without_any_issues() {
	validationRes, err := s.loader.ValidateString(context.TODO(), s.specFixtures["valid"], schema.YAMLSpecFormat, createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(validationRes)
}

func (s *LoaderTestSuite) Test_loads_container_from_input_schema_without_any_issues() {
	container, err := s.loader.LoadFromSchema(context.TODO(), s.specFixtureSchemas["valid"], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(container)
}

func (s *LoaderTestSuite) Test_loads_and_transforms_input_blueprint_without_any_issues() {
	container, err := s.loader.Load(context.TODO(), s.specFixtureFiles[validServerlessBlueprintName], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(container)
}

func (s *LoaderTestSuite) Test_loads_and_transforms_input_blueprint_validating_after_transform() {
	container, err := s.loaderValidateAfterTransform.Load(context.TODO(), s.specFixtureFiles[validServerlessBlueprintName], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(container)
}

func (s *LoaderTestSuite) Test_validates_spec_from_input_schema_without_any_issues() {
	validationRes, err := s.loader.ValidateFromSchema(context.TODO(), s.specFixtureSchemas["valid"], createParams())
	s.Require().NoError(err)
	s.Assert().NotNil(validationRes)
}

func (s *LoaderTestSuite) Test_creates_loader_and_validates_blueprint_with_nil_state_container() {
	// State containers are not used for validation-only use cases for loaders,
	// the (e.g. blueprint language server) so a <nil> value for a state container
	// should be acceptable.
	// A change made when implementing the initial version of the core functions introduced
	// a bug where if the state container was <nil> then the loader would panic when
	// trying to create a new loader and the "core" provider was not provided.
	loaderNoStateContainer := NewDefaultLoader(
		s.providersWithoutCore,
		s.specTransformers,
		/* stateContainer */ nil,
		newFSChildResolver(),
		WithLoaderTransformSpec(true),
		WithLoaderRefChainCollectorFactory(refgraph.NewRefChainCollector),
		WithLoaderLogger(s.logger),
	)

	validationRes, err := loaderNoStateContainer.Validate(
		context.TODO(),
		s.specFixtureFiles["valid"],
		createParams(),
	)
	s.Require().NoError(err)
	s.Assert().NotNil(validationRes)
}

func (s *LoaderTestSuite) Test_creates_loader_and_validates_blueprint_with_stub_resource() {
	// A placeholder template will often be used by host applications to instantiate
	// a blueprint loader in order to call the `Destroy` method of a blueprint container
	// without needing to load the actual blueprint document that was used to deploy
	// the current version of the blueprint instance.
	// This is a workaround for the design decision to tie the `Destroy` method to the blueprint
	// container when it doesn't use any of the data from the loaded blueprint document,
	// having this as a work around means that end users do not have to supply the source document
	// when they want to destroy a blueprint instance.
	validationRes, err := s.loader.Validate(
		context.TODO(),
		s.specFixtureFiles["stub-resource"],
		createParams(),
	)
	s.Require().NoError(err)
	s.Assert().NotNil(validationRes)
}

func (s *LoaderTestSuite) Test_reports_expected_error_when_the_provided_spec_is_invalid() {
	// This is for when the spec is invalid JSON/YAML, as test coverage for specific formats
	// is handled by the schema package, we just need to ensure that the error is reported
	// for either format.
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["invalid-yaml"], createParams())
	s.Require().Error(err)
}

func (s *LoaderTestSuite) Test_reports_expected_error_when_the_provided_spec_fails_schema_specific_validation() {
	// This is for when the spec is valid JSON/YAML, but fails validation against the schema.
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["invalid-schema"], createParams())
	s.Require().Error(err)
}

func (s *LoaderTestSuite) Test_reports_expected_error_when_the_provided_spec_contains_unsupported_variable_types() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["unsupported-var-type"], createParams())
	s.Require().Error(err)
}

func (s *LoaderTestSuite) Test_reports_expected_error_when_transform_is_missing() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["missing-transform"], createParams())
	s.Require().Error(err)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_cyclic_references() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["cyclic-ref"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeReferenceCycle,
		loadErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_cyclic_link_and_reference() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["cyclic-ref-2"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeReferenceCycle,
		loadErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_hard_cyclic_link() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["cyclic-ref-3"], createParams())
	s.Require().Error(err)
	linkErr, isLinkErr := err.(*links.LinkError)
	s.Assert().True(isLinkErr)
	s.Assert().Equal(
		links.LinkErrorReasonCodeCircularLinks,
		linkErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_indirect_cyclic_link() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["cyclic-ref-4"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeReferenceCycle,
		loadErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_indirect_cyclic_link_with_explicit_dependency() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["cyclic-ref-5"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeReferenceCycle,
		loadErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_indirect_cyclic_link_via_link_func_arg() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["cyclic-ref-6"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeReferenceCycle,
		loadErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_invalid_resource_each_dependency_1() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["invalid-resource-each-dep-1"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeEachResourceDependency,
		loadErr.ReasonCode,
	)
}

func (s *LoaderTestSuite) Test_reports_error_for_blueprint_with_invalid_resource_each_dependency_2() {
	_, err := s.loader.Load(context.TODO(), s.specFixtureFiles["invalid-resource-each-dep-2"], createParams())
	s.Require().Error(err)
	loadErr, isLoadErr := internal.UnpackLoadError(err)
	s.Assert().True(isLoadErr)
	s.Assert().Equal(
		validation.ErrorReasonCodeEachChildDependency,
		loadErr.ReasonCode,
	)
}

func TestLoaderTestSuite(t *testing.T) {
	suite.Run(t, new(LoaderTestSuite))
}
