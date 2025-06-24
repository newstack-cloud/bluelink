package provider

import (
	"context"
	"slices"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/errors"
	. "gopkg.in/check.v1"
)

type DataSourceRegistryTestSuite struct {
	dataSourceRegistry DataSourceRegistry
	testDataSource     *testExampleDataSource
}

var _ = Suite(&DataSourceRegistryTestSuite{})

func (s *DataSourceRegistryTestSuite) SetUpTest(c *C) {
	testDataSource := newTestExampleDataSource(
		/* emulateTransientFailures */ true,
	)

	providers := map[string]Provider{
		"test": &testProvider{
			dataSources: map[string]DataSource{
				"test/exampleDataSource": testDataSource,
			},
			namespace: "test",
		},
	}

	s.testDataSource = testDataSource.(*testExampleDataSource)
	s.dataSourceRegistry = NewDataSourceRegistry(
		providers,
		core.SystemClock{},
		core.NewNopLogger(),
	)
}

func (s *DataSourceRegistryTestSuite) Test_get_definition(c *C) {
	output, err := s.dataSourceRegistry.GetSpecDefinition(
		context.TODO(),
		"test/exampleDataSource",
		&DataSourceGetSpecDefinitionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.SpecDefinition, DeepEquals, s.testDataSource.definition)

	// Second time should be cached and produce the same result.
	output, err = s.dataSourceRegistry.GetSpecDefinition(
		context.TODO(),
		"test/exampleDataSource",
		&DataSourceGetSpecDefinitionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.SpecDefinition, DeepEquals, s.testDataSource.definition)
}

func (s *DataSourceRegistryTestSuite) Test_get_type_description(c *C) {
	output, err := s.dataSourceRegistry.GetTypeDescription(
		context.TODO(),
		"test/exampleDataSource",
		&DataSourceGetTypeDescriptionInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.MarkdownDescription, Equals, s.testDataSource.markdownDescription)
	c.Assert(output.PlainTextDescription, Equals, s.testDataSource.plainTextDescription)
}

func (s *DataSourceRegistryTestSuite) Test_get_filter_fields(c *C) {
	output, err := s.dataSourceRegistry.GetFilterFields(
		context.TODO(),
		"test/exampleDataSource",
		&DataSourceGetFilterFieldsInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.FilterFields, DeepEquals, s.testDataSource.filterFields)
}

func (s *DataSourceRegistryTestSuite) Test_custom_validate(c *C) {
	output, err := s.dataSourceRegistry.CustomValidate(
		context.TODO(),
		"test/exampleDataSource",
		&DataSourceValidateInput{},
	)
	c.Assert(err, IsNil)
	c.Assert(output.Diagnostics, DeepEquals, []*core.Diagnostic{
		{
			Level:   core.DiagnosticLevelError,
			Message: "This is a test diagnostic.",
		},
	})
}

func (s *DataSourceRegistryTestSuite) Test_fetch(c *C) {
	output, err := s.dataSourceRegistry.Fetch(
		context.TODO(),
		"test/exampleDataSource",
		&DataSourceFetchInput{},
	)
	c.Assert(err, IsNil)
	expectedName := "test"
	c.Assert(output.Data, DeepEquals, map[string]*core.MappingNode{
		"name": {
			Scalar: &core.ScalarValue{
				StringValue: &expectedName,
			},
		},
	})
}

func (s *DataSourceRegistryTestSuite) Test_has_data_source_type(c *C) {
	hasDSType, err := s.dataSourceRegistry.HasDataSourceType(context.TODO(), "test/exampleDataSource")
	c.Assert(err, IsNil)
	c.Assert(hasDSType, Equals, true)

	hasDSType, err = s.dataSourceRegistry.HasDataSourceType(context.TODO(), "test/otherDataSource")
	c.Assert(err, IsNil)
	c.Assert(hasDSType, Equals, false)
}

func (s *DataSourceRegistryTestSuite) Test_list_data_source_types(c *C) {
	dataSourceTypes, err := s.dataSourceRegistry.ListDataSourceTypes(
		context.TODO(),
	)
	c.Assert(err, IsNil)

	containsTestExampleDataSource := slices.Contains(
		dataSourceTypes,
		"test/exampleDataSource",
	)
	c.Assert(containsTestExampleDataSource, Equals, true)

	// Second time should be cached and produce the same result.
	dataSourceTypesCached, err := s.dataSourceRegistry.ListDataSourceTypes(
		context.TODO(),
	)
	c.Assert(err, IsNil)

	containsCachedTestExampleDataSource := slices.Contains(
		dataSourceTypesCached,
		"test/exampleDataSource",
	)
	c.Assert(containsCachedTestExampleDataSource, Equals, true)
}

func (s *DataSourceRegistryTestSuite) Test_produces_error_for_missing_provider(c *C) {
	_, err := s.dataSourceRegistry.HasDataSourceType(context.TODO(), "otherProvider/otherDataSource")
	c.Assert(err, NotNil)
	runErr, isRunErr := err.(*errors.RunError)
	c.Assert(isRunErr, Equals, true)
	c.Assert(runErr.ReasonCode, Equals, ErrorReasonCodeItemTypeProviderNotFound)
	c.Assert(runErr.Error(), Equals, "run error: run failed as the provider with namespace \"otherProvider\" "+
		"was not found for data source type \"otherProvider/otherDataSource\"")
}
