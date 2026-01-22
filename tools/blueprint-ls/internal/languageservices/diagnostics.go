package languageservices

import (
	"fmt"
	"os"

	"github.com/davecgh/go-spew/spew"
	"github.com/newstack-cloud/bluelink/libs/blueprint/container"
	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/blueprint"
	"github.com/newstack-cloud/bluelink/tools/blueprint-ls/internal/diagnostichelpers"
	"github.com/newstack-cloud/ls-builder/common"
	lsp "github.com/newstack-cloud/ls-builder/lsp_3_17"
	"go.uber.org/zap"
)

// DiagnosticsService is a service that provides functionality
// for diagnostics.
type DiagnosticsService struct {
	state                  *State
	settingsService        *SettingsService
	diagnosticErrorService *DiagnosticErrorService
	loader                 container.Loader
	logger                 *zap.Logger
}

// NewDiagnosticsService creates a new service for diagnostics.
func NewDiagnosticsService(
	state *State,
	settingsService *SettingsService,
	diagnosticErrorService *DiagnosticErrorService,
	loader container.Loader,
	logger *zap.Logger,
) *DiagnosticsService {
	return &DiagnosticsService{
		state,
		settingsService,
		diagnosticErrorService,
		loader,
		logger,
	}
}

// UpdateLoader updates the blueprint loader used by the diagnostics service.
// This is called after plugin loading to use a loader with plugin providers.
func (s *DiagnosticsService) UpdateLoader(loader container.Loader) {
	s.loader = loader
}

// ValidateTextDocument validates a text document and returns diagnostics.
func (s *DiagnosticsService) ValidateTextDocument(
	lspCtx *common.LSPContext,
	docURI lsp.URI,
) ([]lsp.Diagnostic, *schema.Blueprint, error) {
	diagnostics := []lsp.Diagnostic{}
	settings, err := s.settingsService.GetDocumentSettings(lspCtx, docURI)
	if err != nil {
		return nil, nil, err
	}
	s.logger.Debug(fmt.Sprintf("Settings: %v", settings))
	content := s.state.GetDocumentContent(docURI)
	if content == nil {
		return diagnostics, nil, nil
	}

	format := blueprint.DetermineDocFormat(docURI)
	validationResult, err := s.loader.ValidateString(
		safeContext(lspCtx),
		*content,
		format,
		core.NewDefaultParams(
			map[string]map[string]*core.ScalarValue{},
			map[string]map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{},
			map[string]*core.ScalarValue{},
		),
	)
	s.logger.Info("Blueprint diagnostics: ")
	spew.Fdump(os.Stderr, validationResult.Diagnostics)
	diagnostics = append(
		diagnostics,
		diagnostichelpers.BlueprintToLSP(
			validationResult.Diagnostics,
		)...,
	)
	if err != nil {
		s.logger.Error(fmt.Sprintf("Error loading blueprint: %v", err))
		errDiagnostics := s.diagnosticErrorService.BlueprintErrorToDiagnostics(
			err,
			docURI,
		)
		diagnostics = append(diagnostics, errDiagnostics...)
	}

	return deduplicateDiagnostics(diagnostics), validationResult.Schema, nil
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
