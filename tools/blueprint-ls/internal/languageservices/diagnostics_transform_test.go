package languageservices

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/transform"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/deployconfig"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

const transformTestBlueprint = `
version: "2025-11-02"
transform: test-transform
resources:
  sourceResource:
    type: test/abstract
    spec:
      name: source
`

// TransformDiagnosticsSuite covers validation of documents whose blueprints are
// expanded by a transformer.
type TransformDiagnosticsSuite struct {
	suite.Suite
	logger *zap.Logger
}

func (s *TransformDiagnosticsSuite) SetupTest() {
	s.logger = zap.NewNop()
}

// stubTransformer stands in for a transformer plugin. It reports a diagnostic
// and returns a blueprint that differs from its input, mirroring how a real
// transformer replaces abstract resources with concrete ones.
type stubTransformer struct{}

func (t *stubTransformer) GetTransformName(ctx context.Context) (string, error) {
	return "test-transform", nil
}

func (t *stubTransformer) ConfigDefinition(ctx context.Context) (*core.ConfigDefinition, error) {
	return &core.ConfigDefinition{}, nil
}

func (t *stubTransformer) Transform(
	ctx context.Context,
	input *transform.SpecTransformerTransformInput,
) (*transform.SpecTransformerTransformOutput, error) {
	return &transform.SpecTransformerTransformOutput{
		// Deliberately omits the version and renames the resource, as a real
		// transformer's output is a generated blueprint rather than the source.
		TransformedBlueprint: &schema.Blueprint{
			Resources: &schema.ResourceMap{
				Values: map[string]*schema.Resource{
					"generatedResource": {},
				},
			},
		},
		Diagnostics: []*core.Diagnostic{
			{
				Level:   core.DiagnosticLevelWarning,
				Message: "transform stage diagnostic",
			},
		},
	}, nil
}

func (t *stubTransformer) ValidateLinks(
	ctx context.Context,
	input *transform.SpecTransformerValidateLinksInput,
) (*transform.SpecTransformerValidateLinksOutput, error) {
	return &transform.SpecTransformerValidateLinksOutput{}, nil
}

func (t *stubTransformer) AbstractResource(
	ctx context.Context,
	resourceType string,
) (transform.AbstractResource, error) {
	return nil, nil
}

func (t *stubTransformer) ListAbstractResourceTypes(ctx context.Context) ([]string, error) {
	return []string{"test/abstract"}, nil
}

func (t *stubTransformer) ListAbstractLinkTypes(ctx context.Context) ([]string, error) {
	return []string{}, nil
}

func (t *stubTransformer) AbstractLink(
	ctx context.Context,
	linkType string,
) (transform.AbstractLink, error) {
	return nil, nil
}

func (s *TransformDiagnosticsSuite) newService(deployConfigDir string) (*DiagnosticsService, lsp.URI) {
	return s.newServiceForDocument(deployConfigDir, "app.blueprint.yaml", transformTestBlueprint)
}

func (s *TransformDiagnosticsSuite) newServiceForDocument(
	deployConfigDir string,
	fileName string,
	content string,
) (*DiagnosticsService, lsp.URI) {
	state := NewState()
	docURI := lsp.URI("file://" + filepath.Join(deployConfigDir, fileName))
	state.SetDocumentContent(docURI, content)

	transformers := map[string]transform.SpecTransformer{"test-transform": &stubTransformer{}}
	newLoader := func(transformSpec bool) container.Loader {
		return container.NewDefaultLoader(
			nil,
			transformers,
			nil,
			nil,
			container.WithLoaderValidateRuntimeValues(false),
			container.WithLoaderTransformSpec(transformSpec),
		)
	}

	service := NewDiagnosticsService(
		state,
		NewSettingsService(state, "blueprintLanguageServer", s.logger),
		NewDiagnosticErrorService(state, s.logger),
		Loaders{
			WithTransform:    newLoader(true),
			WithoutTransform: newLoader(false),
			TransformEnabled: true,
		},
		deployconfig.NewResolver(deployconfig.ResolverConfig{}, s.logger),
		s.logger,
	)

	return service, docURI
}

func (s *TransformDiagnosticsSuite) writeDeployConfig(dir string) {
	config := `{"appName":"transform-test","deployTarget":{"name":"aws-serverless"}}`
	s.Require().NoError(
		os.WriteFile(filepath.Join(dir, "app.deploy.jsonc"), []byte(config), 0644),
	)
}

// Document structures are built from the returned schema, and every
// position-based feature maps back to the source document, so the transformed
// blueprint must never be handed back to the caller.
func (s *TransformDiagnosticsSuite) Test_returns_source_schema_when_transforming() {
	dir := s.T().TempDir()
	s.writeDeployConfig(dir)
	service, docURI := s.newService(dir)

	_, _, resultSchema, err := service.ValidateTextDocumentBackground(docURI)

	s.Require().NoError(err)
	s.Require().NotNil(resultSchema)
	s.Require().NotNil(resultSchema.Version, "source schema retains its version")
	s.Require().NotNil(resultSchema.Resources)
	s.Contains(resultSchema.Resources.Values, "sourceResource")
	s.NotContains(resultSchema.Resources.Values, "generatedResource")
}

// The blueprint language format has a separate parser, so reparsing the source
// has to dispatch on format. Getting this wrong silently drops the schema and
// takes hover, symbols and go-to-definition down with it.
func (s *TransformDiagnosticsSuite) Test_returns_source_schema_for_blueprint_lang_documents() {
	dir := s.T().TempDir()
	s.writeDeployConfig(dir)
	service, docURI := s.newServiceForDocument(dir, "app.blueprint", `
version "2025-11-02"

transform "test-transform"

resource sourceResource: test/abstract {
    spec {
        name = "source"
    }
}
`)

	_, _, resultSchema, err := service.ValidateTextDocumentBackground(docURI)

	s.Require().NoError(err)
	s.Require().NotNil(resultSchema, "blueprint language documents must still yield a schema")
	s.Require().NotNil(resultSchema.Resources)
	s.Contains(resultSchema.Resources.Values, "sourceResource")
	s.NotContains(resultSchema.Resources.Values, "generatedResource")
}

func (s *TransformDiagnosticsSuite) Test_reports_transform_stage_diagnostics() {
	dir := s.T().TempDir()
	s.writeDeployConfig(dir)
	service, docURI := s.newService(dir)

	diagnostics, _, _, err := service.ValidateTextDocumentBackground(docURI)

	s.Require().NoError(err)
	s.Contains(messagesOf(diagnostics), "transform stage diagnostic")
}

// Without deploy configuration the transformer has no deploy target to emit
// for, so it must not run at all.
func (s *TransformDiagnosticsSuite) Test_skips_transform_without_deploy_config() {
	service, docURI := s.newService(s.T().TempDir())

	diagnostics, _, resultSchema, err := service.ValidateTextDocumentBackground(docURI)

	s.Require().NoError(err)
	s.NotContains(messagesOf(diagnostics), "transform stage diagnostic")
	s.Require().NotNil(resultSchema)
	s.Contains(resultSchema.Resources.Values, "sourceResource")
}

func (s *TransformDiagnosticsSuite) Test_warns_when_deploy_config_is_unusable() {
	dir := s.T().TempDir()
	s.Require().NoError(os.WriteFile(
		filepath.Join(dir, "app.deploy.jsonc"),
		[]byte(`{"appName":"bad","deployTarget":{"name":"nonexistent-target"}}`),
		0644,
	))
	service, docURI := s.newService(dir)

	diagnostics, _, _, err := service.ValidateTextDocumentBackground(docURI)

	s.Require().NoError(err)
	s.Contains(joinMessages(diagnostics), "could not be used")
	s.NotContains(messagesOf(diagnostics), "transform stage diagnostic")
}

func messagesOf(diagnostics []lsp.Diagnostic) []string {
	messages := make([]string, 0, len(diagnostics))
	for _, diagnostic := range diagnostics {
		messages = append(messages, diagnostic.Message)
	}
	return messages
}

func joinMessages(diagnostics []lsp.Diagnostic) string {
	joined := ""
	for _, diagnostic := range diagnostics {
		joined += diagnostic.Message + "\n"
	}
	return joined
}

func TestTransformDiagnosticsSuite(t *testing.T) {
	suite.Run(t, new(TransformDiagnosticsSuite))
}
