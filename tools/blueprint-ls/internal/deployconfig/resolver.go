package deployconfig

import (
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"sync"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"go.uber.org/zap"
)

// Backstop on the upward search, for a blueprint with no discoverable project
// boundary above it.
//
// Boundaries do the real work of stopping the walk; this only limits how far it
// runs when there is none to find. A blueprint sits a handful of directories
// below its project root even in a monorepo, so a walk that gets this far has
// left the project behind and will not find its configuration.
const maxWalkDepth = 8

// Bounds the search when no workspace root is known. A repository root is the
// best available stand-in for a project root. A client may send no workspace
// folders at all, and without some boundary the walk continues into a home
// directory, where an unrelated deploy configuration would be picked up and
// silently applied.
const repositoryMarker = ".git"

// Result is the outcome of resolving deploy configuration for a document.
type Result struct {
	// Resolved is the configuration that was found, or nil when the document
	// has no deploy configuration anywhere above it.
	Resolved *Resolved
	// Err is set when a configuration file was found but could not be used.
	// The distinction matters as a missing file is normal and silent, whereas an
	// unusable one is worth reporting to the user.
	Err error
	// SourcePath is the file that was found, set even when Err is non-nil.
	SourcePath string
}

// Usable reports whether transformer plugins can be run for this result.
// Running them without deploy configuration makes transformers fail on an
// empty deploy target, which is less useful than not transforming at all.
func (r *Result) Usable() bool {
	return r != nil && r.Resolved != nil && r.Err == nil
}

// Params returns the blueprint parameters for this result, falling back to
// bare validation parameters when no configuration is usable.
func (r *Result) Params() core.BlueprintParams {
	if !r.Usable() {
		return ValidationParams()
	}
	return r.Resolved.Params()
}

type cacheEntry struct {
	result   *Result
	fileSize int64
	modTime  int64
}

// Resolver discovers deploy configuration for blueprint documents.
type Resolver struct {
	sources        []Source
	explicitPath   string
	candidatePaths []string
	workspaceRoots []string
	cache          map[string]*cacheEntry
	mu             sync.RWMutex
	logger         *zap.Logger
}

// ResolverConfig configures how deploy configuration is located. Both fields are
// optional; the zero value searches the conventional locations.
type ResolverConfig struct {
	// ExplicitPath names a single configuration file to use for every document,
	// taking precedence over any search. An absolute path is used directly; a
	// relative one is resolved against each directory of the upward search, so a
	// per-environment file resolves against the project rather than against
	// whatever directory the server happens to run in.
	ExplicitPath string
	// CandidatePaths replaces the conventional relative paths probed during the
	// search, in preference order. Projects that keep deploy configuration under
	// a name or directory of their own choosing need this rather than a change
	// here. Each path's format is inferred from its file name.
	CandidatePaths []string
	// WorkspaceRoots bound the upward search. The search examines a root but
	// never goes above one, so a configuration file outside the workspace cannot
	// be picked up for a document inside it.
	//
	// When empty, the search falls back to stopping at a repository root.
	WorkspaceRoots []string
}

// NewResolver creates a resolver for the given configuration.
func NewResolver(config ResolverConfig, logger *zap.Logger) *Resolver {
	return &Resolver{
		sources:        DefaultSources(),
		explicitPath:   config.ExplicitPath,
		candidatePaths: config.CandidatePaths,
		workspaceRoots: config.WorkspaceRoots,
		cache:          map[string]*cacheEntry{},
		logger:         logger,
	}
}

// ResolveForDocument resolves the deploy configuration for a document URI.
func (r *Resolver) ResolveForDocument(docURI string) *Result {
	return r.Resolve(parentDirFromURI(docURI))
}

// Resolve resolves the deploy configuration for a directory, searching it and
// each of its parents.
func (r *Resolver) Resolve(dir string) *Result {
	// An absolute configured path is the whole answer, with no search to do.
	if filepath.IsAbs(r.explicitPath) {
		return r.loadPath(r.explicitPath, sourceForPath(r.explicitPath, r.sources))
	}

	if dir == "" {
		return &Result{}
	}

	if cached := r.cachedResult(dir); cached != nil {
		return cached
	}

	result := r.search(dir)
	r.storeResult(dir, result)

	return result
}

// Invalidate clears all cached results, for use when the client reports that a
// deploy configuration file changed.
func (r *Resolver) Invalidate() {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache = map[string]*cacheEntry{}
}

func (r *Resolver) search(dir string) *Result {
	current := dir
	for range maxWalkDepth {
		if result := r.probeDirectory(current); result != nil {
			return result
		}

		// The boundary directory is searched, then the walk stops. Going above it
		// would reach files belonging to a different project, or to no project.
		if r.isBoundary(current) {
			break
		}

		parent := filepath.Dir(current)
		if parent == current {
			break
		}
		current = parent
	}

	return &Result{}
}

// Reports whether the walk should stop after searching a directory.
//
// Configured workspace roots are authoritative when present, since the client
// knows what the user opened. Otherwise a repository root is used, which keeps a
// blueprint opened as a loose file from reaching into a home directory.
func (r *Resolver) isBoundary(dir string) bool {
	if len(r.workspaceRoots) > 0 {
		return slices.Contains(r.workspaceRoots, dir)
	}

	_, err := os.Stat(filepath.Join(dir, repositoryMarker))

	return err == nil
}

// Looks for a deploy configuration in a single directory,
// returning nil when it holds none.
//
// A relative configured path is probed at each level ahead of the conventional
// names. That is what makes a per-environment file work, a project deploying to
// several environments keeps one file per environment and names the one in use,
// so "bluelink.deploy.dev.jsonc" has to resolve against the project rather than
// against whatever directory the server was started in.
func (r *Resolver) probeDirectory(dir string) *Result {
	for _, relativePath := range r.searchPaths() {
		candidate := filepath.Join(dir, relativePath)
		if _, err := os.Stat(candidate); err != nil {
			continue
		}

		return r.loadPath(candidate, sourceForPath(candidate, r.sources))
	}

	return nil
}

// Returns the relative paths to probe in each directory, in
// preference order.
//
// A configured path or candidate list is an explicit instruction, so it replaces
// the conventional names rather than being tried alongside them: a project that
// names its configuration explicitly should not silently pick up a differently
// named file that happens to be nearby.
func (r *Resolver) searchPaths() []string {
	if r.explicitPath != "" {
		return []string{r.explicitPath}
	}

	if len(r.candidatePaths) > 0 {
		return r.candidatePaths
	}

	return DefaultCandidatePaths()
}

func (r *Resolver) loadPath(path string, source Source) *Result {
	config, err := source.Load(path)
	if err != nil {
		r.logger.Warn(
			"Failed to load deploy configuration",
			zap.String("path", path),
			zap.Error(err),
		)
		return &Result{Err: err, SourcePath: path}
	}

	r.logger.Debug("Loaded deploy configuration", zap.String("path", path))

	return &Result{
		Resolved:   &Resolved{Config: config, SourcePath: path},
		SourcePath: path,
	}
}

func (r *Resolver) cachedResult(dir string) *Result {
	r.mu.RLock()
	entry, found := r.cache[dir]
	r.mu.RUnlock()

	if !found {
		return nil
	}

	info, err := os.Stat(entry.result.SourcePath)
	if err != nil ||
		info.Size() != entry.fileSize ||
		info.ModTime().UnixNano() != entry.modTime {
		return nil
	}

	return entry.result
}

// Caches a result that can later be revalidated against the file it
// came from. A result with no file behind it is not cached and the search would
// have to be repeated to notice a configuration file appearing, so caching it
// would only consume memory for an entry that can never be used.
func (r *Resolver) storeResult(dir string, result *Result) {
	if result.SourcePath == "" {
		return
	}

	info, err := os.Stat(result.SourcePath)
	if err != nil {
		return
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	r.cache[dir] = &cacheEntry{
		result:   result,
		fileSize: info.Size(),
		modTime:  info.ModTime().UnixNano(),
	}
}

// FilePathFromURI converts a file URI to a local path, returning an empty string
// for anything that does not name a local file.
func FilePathFromURI(uri string) string {
	parsed, err := url.Parse(uri)
	if err != nil {
		return ""
	}

	if parsed.Scheme != "file" {
		return ""
	}

	return parsed.Path
}

func parentDirFromURI(docURI string) string {
	path := FilePathFromURI(docURI)
	if path == "" {
		return ""
	}

	return filepath.Dir(path)
}
