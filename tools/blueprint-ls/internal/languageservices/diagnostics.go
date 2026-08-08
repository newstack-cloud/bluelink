package languageservices

import (
	"context"
	"fmt"

	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/deployconfig"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/diagnostichelpers"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// Loaders holds the blueprint loaders the diagnostics service selects between
// on a per-document basis.
type Loaders struct {
	// WithTransform runs transformer plugins during validation, expanding
	// abstract resources so transform-stage diagnostics are produced.
	WithTransform container.Loader
	// WithoutTransform skips transformer plugins. It is used when a document
	// has no usable deploy configuration, since transformers resolve their
	// deploy target from it and fail outright when it is missing.
	WithoutTransform container.Loader
	// TransformEnabled reports whether WithTransform actually runs transformer
	// plugins. It is false when the client has turned spec transformation off,
	// in which case both loaders behave identically.
	TransformEnabled bool
}

// DiagnosticsService is a service that provides functionality
// for diagnostics.
type DiagnosticsService struct {
	state                  *State
	settingsService        *SettingsService
	diagnosticErrorService *DiagnosticErrorService
	loaders                Loaders
	deployConfigResolver   *deployconfig.Resolver
	showAnyTypeWarnings    bool
	logger                 *zap.Logger
}

// NewDiagnosticsService creates a new service for diagnostics.
func NewDiagnosticsService(
	state *State,
	settingsService *SettingsService,
	diagnosticErrorService *DiagnosticErrorService,
	loaders Loaders,
	deployConfigResolver *deployconfig.Resolver,
	logger *zap.Logger,
) *DiagnosticsService {
	return &DiagnosticsService{
		state:                  state,
		settingsService:        settingsService,
		diagnosticErrorService: diagnosticErrorService,
		loaders:                loaders,
		deployConfigResolver:   deployConfigResolver,
		showAnyTypeWarnings:    true,
		logger:                 logger,
	}
}

// SetShowAnyTypeWarnings configures whether "any" type warnings are included
// in published diagnostics.
func (s *DiagnosticsService) SetShowAnyTypeWarnings(show bool) {
	s.showAnyTypeWarnings = show
}

// UpdateLoaders updates the blueprint loaders used by the diagnostics service.
// This is called after plugin loading to use loaders with plugin providers.
func (s *DiagnosticsService) UpdateLoaders(loaders Loaders) {
	s.loaders = loaders
}

// UpdateDeployConfigResolver updates the resolver used to find deploy
// configuration for documents, applying options sent by the client.
func (s *DiagnosticsService) UpdateDeployConfigResolver(resolver *deployconfig.Resolver) {
	s.deployConfigResolver = resolver
}

// InvalidateDeployConfig clears cached deploy configuration, for use when the
// client reports that a deploy configuration file changed.
func (s *DiagnosticsService) InvalidateDeployConfig() {
	s.deployConfigResolver.Invalidate()
}

// ValidateTextDocument validates a text document and returns diagnostics.
// It returns both standard LSP diagnostics and enhanced diagnostics with
// error context metadata for use in code actions.
func (s *DiagnosticsService) ValidateTextDocument(
	lspCtx *common.LSPContext,
	docURI lsp.URI,
) ([]lsp.Diagnostic, []*EnhancedDiagnostic, *schema.Blueprint, error) {
	settings, err := s.settingsService.GetDocumentSettings(lspCtx, docURI)
	if err != nil {
		return nil, nil, nil, err
	}
	s.logger.Debug(fmt.Sprintf("Settings: %v", settings))

	return s.validate(lspCtx.Context, docURI)
}

func (s *DiagnosticsService) validate(
	ctx context.Context,
	docURI lsp.URI,
) ([]lsp.Diagnostic, []*EnhancedDiagnostic, *schema.Blueprint, error) {
	diagnostics := []lsp.Diagnostic{}
	enhanced := []*EnhancedDiagnostic{}

	content := s.state.GetDocumentContent(docURI)
	if content == nil {
		return diagnostics, enhanced, nil, nil
	}

	// Check for duplicate keys from the DocumentContext (parsed AST)
	docCtx := s.state.GetDocumentContext(string(docURI))
	if docCtx != nil && docCtx.DuplicateKeys != nil {
		duplicateDiags := DuplicateKeysToDiagnostics(docCtx.DuplicateKeys)
		diagnostics = append(diagnostics, duplicateDiags...)
	}

	deployConfig := s.deployConfigResolver.ResolveForDocument(string(docURI))
	if deployConfig.Err != nil {
		diagnostics = append(diagnostics, deployConfigDiagnostic(deployConfig))
	}

	format := blueprint.DetermineDocFormat(docURI)
	transforming := deployConfig.Usable() && s.loaders.TransformEnabled
	validationResult, err := s.loaderFor(deployConfig).ValidateString(
		ctx,
		*content,
		format,
		deployConfig.Params(),
	)
	diagnostics = append(
		diagnostics,
		diagnostichelpers.BlueprintToLSP(
			validationResult.Diagnostics,
			s.showAnyTypeWarnings,
		)...,
	)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Error loading blueprint: %v", err))
		errDiagnostics, errEnhanced := s.diagnosticErrorService.BlueprintErrorToDiagnostics(
			err,
			docURI,
		)
		diagnostics = append(diagnostics, errDiagnostics...)
		enhanced = append(enhanced, errEnhanced...)
	}

	return deduplicateDiagnostics(diagnostics),
		enhanced,
		s.documentSchema(validationResult, transforming, *content, format),
		nil
}

// Returns the schema the caller should derive document
// structures from.
//
// When transformers run, the loader reports the transformed blueprint, which
// describes generated concrete resources rather than what the user wrote. Hover,
// symbols and go-to-definition all map positions back to the source document, so
// they must be built from the source blueprint instead. It is reparsed here
// rather than validated a second time, since only its shape is needed.
func (s *DiagnosticsService) documentSchema(
	validationResult *container.ValidationResult,
	transforming bool,
	content string,
	format schema.SpecFormat,
) *schema.Blueprint {
	if !transforming {
		return validationResult.Schema
	}

	sourceSchema, err := blueprint.LoadSchemaString(content, format)
	if err != nil {
		// The document does not parse, so there are no structures to derive.
		// Callers preserve the last known good schema in this case.
		s.logger.Debug(fmt.Sprintf("Failed to reparse source blueprint: %v", err))
		return nil
	}

	return sourceSchema
}

func (s *DiagnosticsService) loaderFor(deployConfig *deployconfig.Result) container.Loader {
	if deployConfig.Usable() {
		return s.loaders.WithTransform
	}

	return s.loaders.WithoutTransform
}

// Reports a deploy configuration file that was found but
// could not be used. Staying silent here would leave the user with quietly
// reduced validation and no indication why.
func deployConfigDiagnostic(deployConfig *deployconfig.Result) lsp.Diagnostic {
	severity := lsp.DiagnosticSeverityWarning
	source := "blueprint-validator"

	return lsp.Diagnostic{
		Range:    lsp.Range{Start: lsp.Position{Line: 0, Character: 0}, End: lsp.Position{Line: 0, Character: 0}},
		Severity: &severity,
		Source:   &source,
		Message: fmt.Sprintf(
			"Deploy configuration at %s could not be used, so transformer-driven validation "+
				"is disabled for this blueprint: %s",
			deployConfig.SourcePath,
			deployConfig.Err,
		),
	}
}

func deduplicateDiagnostics(diagnostics []lsp.Diagnostic) []lsp.Diagnostic {
	if len(diagnostics) == 0 {
		return diagnostics
	}

	seen := make(map[string]bool)
	result := make([]lsp.Diagnostic, 0, len(diagnostics))

	for _, diag := range diagnostics {
		key := diagnosticKey(diag)
		if !seen[key] {
			seen[key] = true
			result = append(result, diag)
		}
	}

	return result
}

func diagnosticKey(diag lsp.Diagnostic) string {
	severity := 0
	if diag.Severity != nil {
		severity = int(*diag.Severity)
	}
	return fmt.Sprintf(
		"%d:%d-%d:%d|%d|%s",
		diag.Range.Start.Line,
		diag.Range.Start.Character,
		diag.Range.End.Line,
		diag.Range.End.Character,
		severity,
		diag.Message,
	)
}

// ValidateTextDocumentBackground validates a document without requiring an LSPContext.
// This is used for debounced validation where the original request context may be cancelled.
// It uses cached settings or defaults instead of making RPC calls to the client.
func (s *DiagnosticsService) ValidateTextDocumentBackground(
	docURI lsp.URI,
) ([]lsp.Diagnostic, []*EnhancedDiagnostic, *schema.Blueprint, error) {
	return s.validate(context.Background(), docURI)
}
