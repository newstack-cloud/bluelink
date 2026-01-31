package languageservices

import (
	"net/url"
	"path/filepath"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/newstack-cloud/bluelink/libs/blueprint/schema"
	"github.com/newstack-cloud/bluelink/libs/blueprint/validation"
	"go.uber.org/zap"
)

// ChildExportInfo holds export information for a single child blueprint export.
type ChildExportInfo struct {
	Name        string
	Type        schema.ExportType
	Description string
	Field       string
}

// ChildBlueprintInfo holds the resolved child blueprint and its export information.
type ChildBlueprintInfo struct {
	Blueprint *schema.Blueprint
	Exports   map[string]*ChildExportInfo
	FilePath  string
}

// ChildBlueprintResolver resolves child blueprint include paths, loads and parses
// child blueprints from disk, caches the results, and provides export info for
// completions and future hover features.
type ChildBlueprintResolver struct {
	cache  *core.Cache[*ChildBlueprintInfo]
	logger *zap.Logger
}

// NewChildBlueprintResolver creates a new resolver for child blueprints.
func NewChildBlueprintResolver(logger *zap.Logger) *ChildBlueprintResolver {
	return &ChildBlueprintResolver{
		cache:  core.NewCache[*ChildBlueprintInfo](),
		logger: logger,
	}
}

// ResolveIncludePath resolves an include path to an absolute filesystem path.
// Returns an empty string if the path cannot be resolved (remote includes,
// unresolvable substitutions, or missing parent directory).
func (r *ChildBlueprintResolver) ResolveIncludePath(
	parentDocURI string,
	include *schema.Include,
) string {
	if include == nil || include.Path == nil {
		return ""
	}

	if validation.IsRemoteInclude(include) {
		return ""
	}

	parentDir := parentDirFromURI(parentDocURI)
	resolveWorkingDir := func() (string, error) {
		return parentDir, nil
	}

	resolvedPath, ok := validation.TryResolveIncludePath(include.Path, resolveWorkingDir)
	if !ok {
		return ""
	}

	if !filepath.IsAbs(resolvedPath) {
		if parentDir == "" {
			return ""
		}
		resolvedPath = filepath.Join(parentDir, resolvedPath)
	}

	return resolvedPath
}

// ResolveChildExports resolves and returns the export info for a child blueprint.
// Returns nil if the child blueprint cannot be resolved.
func (r *ChildBlueprintResolver) ResolveChildExports(
	parentDocURI string,
	include *schema.Include,
) *ChildBlueprintInfo {
	resolvedPath := r.ResolveIncludePath(parentDocURI, include)
	if resolvedPath == "" {
		return nil
	}

	if cached, ok := r.cache.Get(resolvedPath); ok {
		return cached
	}

	info := r.loadAndExtractExports(resolvedPath)
	if info != nil {
		r.cache.Set(resolvedPath, info)
	}
	return info
}

// InvalidateByURI removes cached entries that may be affected by a document change.
func (r *ChildBlueprintResolver) InvalidateByURI(uri string) {
	filePath := filePathFromURI(uri)
	if filePath != "" {
		r.cache.Delete(filePath)
	}
}

func (r *ChildBlueprintResolver) loadAndExtractExports(filePath string) *ChildBlueprintInfo {
	format, err := deriveChildBlueprintFormat(filePath)
	if err != nil {
		r.logger.Debug("Could not determine format for child blueprint",
			zap.String("path", filePath),
			zap.Error(err),
		)
		return nil
	}

	bp, err := schema.Load(filePath, format)
	if err != nil {
		r.logger.Debug("Could not load child blueprint for completions",
			zap.String("path", filePath),
			zap.Error(err),
		)
		return nil
	}

	exports := map[string]*ChildExportInfo{}
	if bp.Exports != nil {
		for name, export := range bp.Exports.Values {
			info := &ChildExportInfo{
				Name: name,
			}
			if export.Type != nil {
				info.Type = export.Type.Value
			}
			if export.Description != nil {
				info.Description = getStringOrSubstitutionsValue(export.Description)
			}
			if export.Field != nil && export.Field.StringValue != nil {
				info.Field = *export.Field.StringValue
			}
			exports[name] = info
		}
	}

	return &ChildBlueprintInfo{
		Blueprint: bp,
		Exports:   exports,
		FilePath:  filePath,
	}
}

func parentDirFromURI(uri string) string {
	filePath := filePathFromURI(uri)
	if filePath == "" {
		return ""
	}
	return filepath.Dir(filePath)
}

func filePathFromURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}
	if parsed.Scheme == "file" {
		return parsed.Path
	}
	return uri
}

func fileURIFromPath(filePath string) string {
	return "file://" + filePath
}

func deriveChildBlueprintFormat(filePath string) (schema.SpecFormat, error) {
	if strings.HasSuffix(filePath, ".yml") || strings.HasSuffix(filePath, ".yaml") {
		return schema.YAMLSpecFormat, nil
	}
	if strings.HasSuffix(filePath, ".jsonc") {
		return schema.JWCCSpecFormat, nil
	}
	return "", errUnsupportedChildBlueprintFormat(filePath)
}

func errUnsupportedChildBlueprintFormat(path string) error {
	return &unsupportedFormatError{path: path}
}

type unsupportedFormatError struct {
	path string
}

func (e *unsupportedFormatError) Error() string {
	return "unsupported child blueprint format: " + e.path
}
