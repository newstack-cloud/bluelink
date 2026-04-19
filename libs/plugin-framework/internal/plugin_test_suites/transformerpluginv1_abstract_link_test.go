package plugintestsuites

import (
	"context"

	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/errorsv1"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testtransformer"
	"github.com/newstack-cloud/bluelink/libs/plugin-framework/internal/testutils"
)

const (
	celerityHandlerAPIAbstractLinkType = "celerity/handler::celerity/api"
)

func (s *TransformerPluginV1Suite) Test_list_abstract_link_types() {
	abstractLinkTypes, err := s.transformer.ListAbstractLinkTypes(context.Background())
	s.Require().NoError(err)
	s.Require().Equal(
		[]string{celerityHandlerAPIAbstractLinkType},
		abstractLinkTypes,
	)
}

func (s *TransformerPluginV1Suite) Test_list_abstract_link_types_fails_for_unexpected_host() {
	_, err := s.transformerWrongHost.ListAbstractLinkTypes(context.Background())
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerListAbstractLinkTypes,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_list_abstract_link_types_reports_expected_error_for_failure() {
	_, err := s.failingTransformer.ListAbstractLinkTypes(context.Background())
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred listing abstract link types")
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_type() {
	abstractLink, err := s.transformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	output, err := abstractLink.GetType(
		context.Background(),
		&transform.AbstractLinkGetTypeInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&transform.AbstractLinkGetTypeOutput{
			Type:          celerityHandlerAPIAbstractLinkType,
			ResourceTypeA: "celerity/handler",
			ResourceTypeB: "celerity/api",
		},
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_type_fails_for_unexpected_host() {
	abstractLink, err := s.transformerWrongHost.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetType(
		context.Background(),
		&transform.AbstractLinkGetTypeInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractLinkType,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_type_reports_expected_error_for_failure() {
	abstractLink, err := s.failingTransformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetType(
		context.Background(),
		&transform.AbstractLinkGetTypeInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(err.Error(), "internal error occurred retrieving abstract link type")
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_type_description() {
	abstractLink, err := s.transformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	output, err := abstractLink.GetTypeDescription(
		context.Background(),
		&transform.AbstractLinkGetTypeDescriptionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testtransformer.AbstractLinkHandlerAPITypeDescription(),
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_type_description_fails_for_unexpected_host() {
	abstractLink, err := s.transformerWrongHost.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetTypeDescription(
		context.Background(),
		&transform.AbstractLinkGetTypeDescriptionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractLinkTypeDescription,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_type_description_reports_expected_error_for_failure() {
	abstractLink, err := s.failingTransformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetTypeDescription(
		context.Background(),
		&transform.AbstractLinkGetTypeDescriptionInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred retrieving abstract link type description",
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_annotation_definitions() {
	abstractLink, err := s.transformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	output, err := abstractLink.GetAnnotationDefinitions(
		context.Background(),
		&transform.AbstractLinkGetAnnotationDefinitionsInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		testtransformer.AbstractLinkHandlerAPIAnnotations(),
		output.AnnotationDefinitions,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_annotation_definitions_fails_for_unexpected_host() {
	abstractLink, err := s.transformerWrongHost.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetAnnotationDefinitions(
		context.Background(),
		&transform.AbstractLinkGetAnnotationDefinitionsInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractLinkAnnotationDefinitions,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_annotation_definitions_reports_expected_error_for_failure() {
	abstractLink, err := s.failingTransformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetAnnotationDefinitions(
		context.Background(),
		&transform.AbstractLinkGetAnnotationDefinitionsInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred retrieving abstract link annotation definitions",
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_cardinality() {
	abstractLink, err := s.transformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	output, err := abstractLink.GetCardinality(
		context.Background(),
		&transform.AbstractLinkGetCardinalityInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Require().NoError(err)
	s.Assert().Equal(
		&transform.AbstractLinkGetCardinalityOutput{
			CardinalityA: provider.LinkCardinality{Min: 0, Max: 1},
			CardinalityB: provider.LinkCardinality{Min: 1, Max: 0},
		},
		output,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_cardinality_fails_for_unexpected_host() {
	abstractLink, err := s.transformerWrongHost.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetCardinality(
		context.Background(),
		&transform.AbstractLinkGetCardinalityInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	testutils.AssertInvalidHost(
		err,
		errorsv1.PluginActionTransformerGetAbstractLinkCardinality,
		testWrongHostID,
		&s.Suite,
	)
}

func (s *TransformerPluginV1Suite) Test_abstract_link_get_cardinality_reports_expected_error_for_failure() {
	abstractLink, err := s.failingTransformer.AbstractLink(
		context.Background(),
		celerityHandlerAPIAbstractLinkType,
	)
	s.Require().NoError(err)

	_, err = abstractLink.GetCardinality(
		context.Background(),
		&transform.AbstractLinkGetCardinalityInput{
			TransformerContext: testutils.CreateTestTransformerContext("celerity"),
		},
	)
	s.Assert().Error(err)
	s.Assert().Contains(
		err.Error(),
		"internal error occurred retrieving abstract link cardinality",
	)
}
