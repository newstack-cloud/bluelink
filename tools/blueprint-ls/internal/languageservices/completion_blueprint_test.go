package languageservices

import (
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/lang"
	"github.com/newstack-cloud/bluelink/libs/blueprint/provider"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/docmodel"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/linkinfo"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/testutils"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

func (s *CompletionServiceGetItemsSuite) blueprintObjectService() *CompletionService {
	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)
	state := NewState()
	state.SetLinkSupportCapability(true)
	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/lambda/function": &testutils.LambdaFunctionResource{},
		},
	}
	return NewCompletionService(
		resourceRegistry,
		&testutils.DataSourceRegistryMock{},
		&testutils.CustomVarTypeRegistryMock{},
		&testutils.FunctionRegistryMock{},
		nil, state, logger,
	)
}

func (s *CompletionServiceGetItemsSuite) createBlueprintDocContext(
	validContent string, editingContent string,
) *docmodel.DocumentContext {
	blueprint, err := lang.ParseString(validContent)
	s.Require().NoError(err)
	tree := schema.SchemaToTree(blueprint)
	docCtx := docmodel.NewDocumentContext(blueprintURI, editingContent, docmodel.FormatBlueprintLang, nil)
	docCtx.UpdateSchema(blueprint, tree)
	return docCtx
}

func (s *CompletionServiceGetItemsSuite) bpCompletionLabels(
	docCtx *docmodel.DocumentContext, line, character int,
) []string {
	items, err := s.service.GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: lsp.UInteger(line), Character: lsp.UInteger(character)},
		},
	)
	s.Require().NoError(err)
	return completionItemLabels(items.Items)
}

func containsLabel(labels []string, target string) bool {
	for _, label := range labels {
		if label == target {
			return true
		}
	}
	return false
}

func (s *CompletionServiceGetItemsSuite) Test_bp_top_level_offers_declaration_keywords() {
	docCtx := s.createBlueprintDocContext("version \"2025-11-02\"\n", "")
	labels := s.bpCompletionLabels(docCtx, 0, 0)

	for _, keyword := range []string{"resource", "variable", "value", "data", "include", "export", "metadata", "version", "transform"} {
		s.Assert().True(containsLabel(labels, keyword), "expected declaration keyword %q", keyword)
	}
	// The YAML section keys must NOT appear in a .bp document.
	s.Assert().False(containsLabel(labels, "resources"), "must not offer YAML section key 'resources'")
	s.Assert().False(containsLabel(labels, "variables"), "must not offer YAML section key 'variables'")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_resource_block_fields_exclude_type() {
	valid := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n" +
		"    spec {\n        tableName = \"orders\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n" +
		"    \n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	labels := s.bpCompletionLabels(docCtx, 2, 4)
	for _, field := range []string{"description", "spec", "metadata", "condition", "foreach", "select by label", "dependsOn", "removalPolicy"} {
		s.Assert().True(containsLabel(labels, field), "expected resource field %q", field)
	}
	// `type` lives in the header, not the body.
	s.Assert().False(containsLabel(labels, "type"), "resource body must not offer 'type'")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_variable_block_fields_exclude_type() {
	valid := "version \"2025-11-02\"\nvariable region: string {\n    default = \"us-east-1\"\n}\n"
	editing := "version \"2025-11-02\"\nvariable region: string {\n    \n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	labels := s.bpCompletionLabels(docCtx, 2, 4)
	for _, field := range []string{"default", "description", "allowedValues", "secret"} {
		s.Assert().True(containsLabel(labels, field), "expected variable field %q", field)
	}
	s.Assert().False(containsLabel(labels, "type"), "variable body must not offer 'type'")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_select_by_label_offers_exclude_not_byLabel() {
	valid := "version \"2025-11-02\"\n" +
		"resource fn: aws/dynamodb/table {\n    spec {\n        tableName = \"t\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/dynamodb/table {\n    select by label {\n        \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	labels := s.bpCompletionLabels(docCtx, 3, 8)
	s.Assert().True(containsLabel(labels, "exclude"), "expected 'exclude' in select by label block")
	s.Assert().False(containsLabel(labels, "byLabel"), "must not offer synthetic 'byLabel'")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_spec_fields_insert_with_equals_not_colon() {
	valid := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n    spec {\n        tableName = \"orders\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n    spec {\n        \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	items, err := s.service.GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 3, Character: 8},
		},
	)
	s.Require().NoError(err)
	s.Require().NotEmpty(items.Items)

	foundTableName := false
	for _, item := range items.Items {
		text := extractNewText(item.TextEdit)
		s.Assert().NotContains(text, ": ", "spec field insert must not use YAML colon: %q", text)
		if item.Label == "tableName" {
			foundTableName = true
			s.Assert().Equal("tableName = ", text)
		}
	}
	s.Assert().True(foundTableName, "expected the tableName spec field")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_object_spec_field_inserts_object_literal() {
	valid := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        functionName = \"fn\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	items, err := s.blueprintObjectService().GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 3, Character: 8},
		},
	)
	s.Require().NoError(err)

	foundThroughput := false
	for _, item := range items.Items {
		if item.Label == "throughput" {
			foundThroughput = true
			// Object spec fields are object literals ("field = { }"), not blocks.
			s.Assert().Equal("throughput = {\n\t$0\n}", extractNewText(item.TextEdit))
		}
	}
	s.Assert().True(foundThroughput, "expected the object-typed throughput spec field")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_nested_object_spec_field_inserts_object_literal() {
	valid := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        throughput = {\n            readCapacity = 5\n        }\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        throughput = {\n            \n        }\n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	// Cursor on the blank line inside the nested throughput object (line 5, 0-based 4).
	items, err := s.blueprintObjectService().GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 4, Character: 12},
		},
	)
	s.Require().NoError(err)

	labels := completionItemLabels(items.Items)
	s.Assert().True(containsLabel(labels, "readCapacity"), "expected nested object field readCapacity")
	for _, item := range items.Items {
		text := extractNewText(item.TextEdit)
		s.Assert().NotContains(text, ": ", "nested spec field insert must not use YAML colon: %q", text)
	}
}

func (s *CompletionServiceGetItemsSuite) bpSpecFieldLabels(editing string, line, character int) []string {
	valid := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        functionName = \"fn\"\n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)
	items, err := s.blueprintObjectService().GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: lsp.UInteger(line), Character: lsp.UInteger(character)},
		},
	)
	s.Require().NoError(err)
	return completionItemLabels(items.Items)
}

func (s *CompletionServiceGetItemsSuite) Test_bp_spec_fields_two_level_nested_object() {
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    spec {\n" +
		"        throughput = {\n" +
		"            autoScaling = {\n" +
		"                \n" +
		"            }\n        }\n    }\n}\n"
	labels := s.bpSpecFieldLabels(editing, 5, 16)
	s.Assert().True(containsLabel(labels, "minCapacity"), "expected nested-object field minCapacity")
	s.Assert().True(containsLabel(labels, "maxCapacity"), "expected nested-object field maxCapacity")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_spec_fields_object_inside_array() {
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    spec {\n" +
		"        layers = [\n" +
		"            {\n" +
		"                \n" +
		"            }\n        ]\n    }\n}\n"
	labels := s.bpSpecFieldLabels(editing, 5, 16)
	s.Assert().True(containsLabel(labels, "layerName"), "expected array-element-object field layerName")
	s.Assert().True(containsLabel(labels, "layerVersion"), "expected array-element-object field layerVersion")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_spec_fields_object_inside_map() {
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n" +
		"    spec {\n" +
		"        tags = {\n" +
		"            \"env\" = {\n" +
		"                \n" +
		"            }\n        }\n    }\n}\n"
	labels := s.bpSpecFieldLabels(editing, 5, 16)
	s.Assert().True(containsLabel(labels, "value"), "expected map-value-object field value")
	s.Assert().True(containsLabel(labels, "protected"), "expected map-value-object field protected")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_allowed_value_spec_field_retriggers_suggest() {
	valid := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n    spec {\n        tableName = \"orders\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n    spec {\n        \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	items, err := s.service.GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 3, Character: 8},
		},
	)
	s.Require().NoError(err)

	for _, item := range items.Items {
		switch item.Label {
		case "billingMode": // has AllowedValues in the schema
			s.Require().NotNil(item.Command, "allowed-value field should re-trigger suggest")
			s.Assert().Equal("editor.action.triggerSuggest", item.Command.Command)
		case "tableName": // a plain string field
			s.Assert().Nil(item.Command, "non-allowed-value field must not re-trigger suggest")
		}
	}
}

func (s *CompletionServiceGetItemsSuite) Test_bp_spec_field_value_offers_allowed_values() {
	valid := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n    spec {\n        billingMode = \"PAY_PER_REQUEST\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource ordersTable: aws/dynamodb/table {\n    spec {\n        billingMode = \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	// Cursor right after "        billingMode = " on line 4 (0-based 3).
	labels := s.bpCompletionLabels(docCtx, 3, 22)
	s.Assert().True(containsLabel(labels, "PAY_PER_REQUEST"), "expected allowed value PAY_PER_REQUEST")
	s.Assert().True(containsLabel(labels, "PROVISIONED"), "expected allowed value PROVISIONED")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_boolean_spec_field_retriggers_suggest() {
	valid := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        functionName = \"fn\"\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	items, err := s.blueprintObjectService().GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 3, Character: 8},
		},
	)
	s.Require().NoError(err)

	for _, item := range items.Items {
		switch item.Label {
		case "tracingEnabled": // a boolean field
			s.Require().NotNil(item.Command, "boolean field should re-trigger suggest")
			s.Assert().Equal("editor.action.triggerSuggest", item.Command.Command)
		case "functionName": // a plain string field
			s.Assert().Nil(item.Command, "non-suggestible field must not re-trigger suggest")
		}
	}
}

func (s *CompletionServiceGetItemsSuite) Test_bp_boolean_spec_field_value_offers_true_false() {
	valid := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        tracingEnabled = true\n    }\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"resource fn: aws/lambda/function {\n    spec {\n        tracingEnabled = \n    }\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	// Cursor right after "        tracingEnabled = " on line 4 (0-based 3).
	items, err := s.blueprintObjectService().GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: 3, Character: 25},
		},
	)
	s.Require().NoError(err)
	labels := completionItemLabels(items.Items)
	s.Assert().True(containsLabel(labels, "true"), "expected boolean literal true")
	s.Assert().True(containsLabel(labels, "false"), "expected boolean literal false")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_bare_reference_at_value_position() {
	valid := "version \"2025-11-02\"\n" +
		"variable environment: string {\n    default = \"prod\"\n}\n" +
		"value name: string {\n    value = variables.environment\n}\n"
	editing := "version \"2025-11-02\"\n" +
		"variable environment: string {\n    default = \"prod\"\n}\n" +
		"value name: string {\n    value = variables.\n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	// Cursor right after "    value = variables." on line 6 (0-based 5).
	labels := s.bpCompletionLabels(docCtx, 5, 22)
	s.Assert().True(containsLabel(labels, "environment"), "expected bare variable reference completion")
}

func (s *CompletionServiceGetItemsSuite) Test_bp_value_literals_at_value_position() {
	valid := "version \"2025-11-02\"\nvariable flag: boolean {\n    default = true\n}\n"
	editing := "version \"2025-11-02\"\nvariable flag: boolean {\n    secret = \n}\n"
	docCtx := s.createBlueprintDocContext(valid, editing)

	// Cursor after "    secret = " on line 3 (0-based 2).
	labels := s.bpCompletionLabels(docCtx, 2, 13)
	for _, literal := range []string{"true", "false", "none"} {
		s.Assert().True(containsLabel(labels, literal), "expected literal %q at value position", literal)
	}
}

// Builds a completion service wired with a Lambda ->
// DynamoDB link whose annotation definition carries AllowedValues, so annotation
// key and value completions can be exercised for .bp documents.
func (s *CompletionServiceGetItemsSuite) blueprintAnnotationService() *CompletionService {
	logger, err := zap.NewDevelopment()
	s.Require().NoError(err)
	state := NewState()
	state.SetLinkSupportCapability(true)

	resourceRegistry := &testutils.ResourceRegistryMock{
		Resources: map[string]provider.Resource{
			"aws/dynamodb/table":  &testutils.DynamoDBTableResource{},
			"aws/lambda/function": &testutils.LambdaFunctionResource{},
		},
	}
	dataSourceRegistry := &testutils.DataSourceRegistryMock{}
	customVarTypeRegistry := &testutils.CustomVarTypeRegistryMock{}
	functionRegistry := &testutils.FunctionRegistryMock{}
	linkRegistry := &testutils.LinkRegistryMock{
		Links: map[string]provider.Link{
			"aws/lambda/function::aws/dynamodb/table": &testutils.MockLink{
				AnnotationDefs: map[string]*provider.LinkAnnotationDefinition{
					"aws/lambda/function::aws.lambda.dynamodb.accessType": {
						Name:        "aws.lambda.dynamodb.accessType",
						Label:       "DynamoDB Access Type",
						Type:        core.ScalarTypeString,
						Description: "The type of access the Lambda function has to the DynamoDB table.",
						AllowedValues: []*core.ScalarValue{
							core.ScalarFromString("read"),
							core.ScalarFromString("write"),
							core.ScalarFromString("readwrite"),
						},
					},
				},
			},
		},
	}

	svc := NewCompletionService(
		resourceRegistry, dataSourceRegistry, customVarTypeRegistry,
		functionRegistry, nil, state, logger,
	)
	svc.UpdateRegistries(
		resourceRegistry, dataSourceRegistry, customVarTypeRegistry,
		functionRegistry, linkinfo.NewProviderSource(linkRegistry),
	)
	return svc
}

// Builds a two-resource .bp document (a Lambda function
// that selects a DynamoDB table by label) where the annotations block body is
// replaced by bodyLine. The cursor line is always line 8 (0-based).
func bpAnnotationBlueprint(bodyLine string) string {
	lines := []string{
		`version "2025-11-02"`, // 0
		``,                     // 1
		`resource saveOrderFunction: aws/lambda/function {`, // 2
		`    metadata {`,                     // 3
		`        labels = {`,                 // 4
		`            app = "orders"`,         // 5
		`        }`,                          // 6
		`        annotations = {`,            // 7
		bodyLine,                             // 8 (cursor line)
		`        }`,                          // 9
		`    }`,                              // 10
		`    select by label {`,              // 11
		`        app = "orders"`,             // 12
		`    }`,                              // 13
		`    spec {`,                         // 14
		`        functionName = "saveOrder"`, // 15
		`    }`,                              // 16
		`}`,                                  // 17
		``,                                   // 18
		`resource ordersTable: aws/dynamodb/table {`, // 19
		`    metadata {`,               // 20
		`        labels = {`,           // 21
		`            app = "orders"`,   // 22
		`        }`,                    // 23
		`    }`,                        // 24
		`    spec {`,                   // 25
		`        tableName = "orders"`, // 26
		`    }`,                        // 27
		`}`,                            // 28
	}
	return strings.Join(lines, "\n") + "\n"
}

// Drives a completion request against the annotation-aware
// service, using a complete valid blueprint as the effective schema and the
// given editing content for cursor/AST resolution.
func (s *CompletionServiceGetItemsSuite) bpAnnotationItems(
	editing string, line, character int,
) *CompletionResult {
	valid := bpAnnotationBlueprint(`            "aws.lambda.dynamodb.accessType" = "read"`)
	docCtx := s.createBlueprintDocContext(valid, editing)
	items, err := s.blueprintAnnotationService().GetCompletionItems(
		&common.LSPContext{}, docCtx,
		&lsp.TextDocumentPositionParams{
			TextDocument: lsp.TextDocumentIdentifier{URI: blueprintURI},
			Position:     lsp.Position{Line: lsp.UInteger(line), Character: lsp.UInteger(character)},
		},
	)
	s.Require().NoError(err)
	return items
}

func (s *CompletionServiceGetItemsSuite) Test_bp_annotation_key_offers_link_annotations() {
	// Cursor on the blank line inside `annotations = { }` (line 8, 12-space indent).
	editing := bpAnnotationBlueprint(`            `)
	items := s.bpAnnotationItems(editing, 8, 12)

	labels := completionItemLabels(items.Items)
	s.Assert().True(containsLabel(labels, "aws.lambda.dynamodb.accessType"),
		"expected the link annotation key, got %v", labels)

	for _, item := range items.Items {
		if item.Label == "aws.lambda.dynamodb.accessType" {
			s.Assert().Equal(`"aws.lambda.dynamodb.accessType" = `, extractNewText(item.TextEdit),
				"blueprint annotation key must insert as a quoted assignment")
		}
	}
}

func (s *CompletionServiceGetItemsSuite) Test_bp_annotation_key_partial_dotted_prefix() {
	// Partial dotted key typed, no "=" yet (line 8: "            aws.lam").
	editing := bpAnnotationBlueprint(`            aws.lam`)
	items := s.bpAnnotationItems(editing, 8, 19)

	labels := completionItemLabels(items.Items)
	s.Assert().True(containsLabel(labels, "aws.lambda.dynamodb.accessType"),
		"expected the link annotation key for a partial dotted prefix, got %v", labels)
}

func (s *CompletionServiceGetItemsSuite) Test_bp_annotation_key_unclosed_object() {
	// Truncated, in-progress edit: the annotations object is never closed.
	editing := strings.Join([]string{
		`version "2025-11-02"`,
		``,
		`resource saveOrderFunction: aws/lambda/function {`,
		`    metadata {`,
		`        labels = {`,
		`            app = "orders"`,
		`        }`,
		`        annotations = {`,
		`            `,
	}, "\n")
	items := s.bpAnnotationItems(editing, 8, 12)

	labels := completionItemLabels(items.Items)
	s.Assert().True(containsLabel(labels, "aws.lambda.dynamodb.accessType"),
		"expected the link annotation key inside an unclosed object, got %v", labels)
}

func (s *CompletionServiceGetItemsSuite) Test_bp_annotation_value_quoted_key() {
	// `"aws.lambda.dynamodb.accessType" = ` with the cursor at the value position.
	editing := bpAnnotationBlueprint(`            "aws.lambda.dynamodb.accessType" = `)
	items := s.bpAnnotationItems(editing, 8, 47)

	labels := completionItemLabels(items.Items)
	for _, value := range []string{"read", "write", "readwrite"} {
		s.Assert().True(containsLabel(labels, value),
			"expected allowed annotation value %q, got %v", value, labels)
	}
}

func (s *CompletionServiceGetItemsSuite) Test_bp_annotation_value_bare_key() {
	// Same as above but the dotted key is bare (unquoted).
	editing := bpAnnotationBlueprint(`            aws.lambda.dynamodb.accessType = `)
	items := s.bpAnnotationItems(editing, 8, 45)

	labels := completionItemLabels(items.Items)
	for _, value := range []string{"read", "write", "readwrite"} {
		s.Assert().True(containsLabel(labels, value),
			"expected allowed annotation value %q for a bare key, got %v", value, labels)
	}
}
