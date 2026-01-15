package plugins

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/ProtonMail/go-crypto/openpgp"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/plugins/version"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/registries"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/utils"
)

// InstallStage represents the current stage of plugin installation.
type InstallStage string

const (
	StageResolving   InstallStage = "resolving"
	StageDownloading InstallStage = "downloading"
	StageVerifying   InstallStage = "verifying"
	StageExtracting  InstallStage = "extracting"
	StageComplete    InstallStage = "complete"
)

// InstallStatus represents the result status of a plugin installation.
type InstallStatus int

const (
	StatusInstalled InstallStatus = iota
	StatusSkipped
	StatusFailed
)

// ProgressCallback is called during installation to report progress.
type ProgressCallback func(pluginID *PluginID, stage InstallStage, downloaded, total int64)

// InstallResult contains the result of a plugin installation attempt.
type InstallResult struct {
	PluginID *PluginID
	Status   InstallStatus
	Error    error
}

// InstalledPlugin represents a plugin that has been installed.
type InstalledPlugin struct {
	ID           string    `json:"id"`
	Version      string    `json:"version"`
	RegistryHost string    `json:"registryHost"`
	Shasum       string    `json:"shasum"`
	InstalledAt  time.Time `json:"installedAt"`
}

// PluginManifest tracks all installed plugins.
type PluginManifest struct {
	Plugins map[string]*InstalledPlugin `json:"plugins"`
}

// Manager handles plugin installation, verification, and manifest management.
type Manager struct {
	registryClient  *registries.RegistryClient
	discoveryClient *registries.ServiceDiscoveryClient
	pluginsDir      string
}

// NewManager creates a new plugin manager.
func NewManager(
	registryClient *registries.RegistryClient,
	discoveryClient *registries.ServiceDiscoveryClient,
) *Manager {
	return &Manager{
		registryClient:  registryClient,
		discoveryClient: discoveryClient,
		pluginsDir:      GetPluginsDir(),
	}
}

// NewManagerWithPluginsDir creates a new plugin manager with a custom plugins directory.
func NewManagerWithPluginsDir(
	registryClient *registries.RegistryClient,
	discoveryClient *registries.ServiceDiscoveryClient,
	pluginsDir string,
) *Manager {
	return &Manager{
		registryClient:  registryClient,
		discoveryClient: discoveryClient,
		pluginsDir:      pluginsDir,
	}
}

// GetPluginsDir returns the plugin installation directory.
// Priority: BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH env var > default platform path.
// If BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH contains multiple paths (separated by
// os.PathListSeparator), the first path is used for installation.
//
// Directory structure:
//   - {pluginsDir}/manifest.json - Plugin manifest tracking installed plugins
//   - {pluginsDir}/bin/{namespace}/{name}/{version}/ - Plugin executables
func GetPluginsDir() string {
	if envPath := os.Getenv("BLUELINK_DEPLOY_ENGINE_PLUGIN_PATH"); envPath != "" {
		// Handle multiple paths separated by os.PathListSeparator
		// (: on Unix, ; on Windows)
		paths := filepath.SplitList(envPath)
		if len(paths) > 0 && paths[0] != "" {
			return utils.ExpandEnv(paths[0])
		}
	}

	// Use platform-appropriate default path with env var expansion
	if runtime.GOOS == "windows" {
		return utils.ExpandEnv("%LOCALAPPDATA%\\NewStack\\Bluelink\\engine\\plugins")
	}

	return utils.ExpandEnv("$HOME/.bluelink/engine/plugins")
}

// Installs a single plugin.
func (m *Manager) Install(
	ctx context.Context,
	pluginID *PluginID,
	progressFn ProgressCallback,
) (*InstallResult, error) {
	results, err := m.InstallAll(ctx, []*PluginID{pluginID}, progressFn)
	if err != nil {
		return nil, err
	}
	if len(results) == 0 {
		// No plugins to install - check if already installed
		if installed, _, _ := m.IsInstalled(pluginID); installed {
			return &InstallResult{PluginID: pluginID, Status: StatusSkipped}, nil
		}
		return nil, fmt.Errorf("unexpected empty results")
	}
	return results[0], nil
}

// InstallAll installs multiple plugins with dependency resolution.
// Dependencies are resolved and installed first in topological order.
func (m *Manager) InstallAll(
	ctx context.Context,
	pluginIDs []*PluginID,
	progressFn ProgressCallback,
) ([]*InstallResult, error) {
	// Resolve all dependencies and get topologically sorted install order
	orderedPlugins, err := m.resolveDependencies(ctx, pluginIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve dependencies: %w", err)
	}

	results := make([]*InstallResult, 0, len(orderedPlugins))

	for _, pluginID := range orderedPlugins {
		result := m.installPlugin(ctx, pluginID, progressFn)
		results = append(results, result)

		// If a dependency fails to install, stop processing
		if result.Status == StatusFailed {
			// Mark remaining plugins as failed due to dependency failure
			// but don't add them to results since they weren't attempted
			break
		}
	}

	return results, nil
}

// InstallMissing checks which plugins are not installed and installs them.
func (m *Manager) InstallMissing(
	ctx context.Context,
	pluginIDs []*PluginID,
	progressFn ProgressCallback,
) ([]*InstallResult, error) {
	missing, err := m.GetMissingPlugins(pluginIDs)
	if err != nil {
		return nil, err
	}

	if len(missing) == 0 {
		return nil, nil
	}

	return m.InstallAll(ctx, missing, progressFn)
}

// GetMissingPlugins returns plugins from the list that are not currently installed.
func (m *Manager) GetMissingPlugins(pluginIDs []*PluginID) ([]*PluginID, error) {
	var missing []*PluginID

	for _, pluginID := range pluginIDs {
		installed, _, err := m.IsInstalled(pluginID)
		if err != nil {
			return nil, err
		}
		if !installed {
			missing = append(missing, pluginID)
		}
	}

	return missing, nil
}

// IsInstalled checks if a plugin is installed with the specified version.
func (m *Manager) IsInstalled(pluginID *PluginID) (bool, *InstalledPlugin, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return false, nil, err
	}

	key := pluginID.ManifestKey()
	installed, exists := manifest.Plugins[key]
	if !exists {
		return false, nil, nil
	}

	// If version is specified, check if it matches
	if pluginID.Version != "" && installed.Version != pluginID.Version {
		return false, nil, nil
	}

	return true, installed, nil
}

func (m *Manager) installPlugin(
	ctx context.Context,
	pluginID *PluginID,
	progressFn ProgressCallback,
) *InstallResult {
	result := &InstallResult{PluginID: pluginID}

	// Check if already installed
	if skipResult, err := m.checkIfAlreadyInstalled(pluginID); err != nil {
		return failedResult(result, err)
	} else if skipResult != nil {
		return skipResult
	}

	// Stage: Resolving
	reportProgress(progressFn, pluginID, StageResolving, 0, 0)

	resolvedID, err := m.resolvePluginVersion(ctx, pluginID)
	if err != nil {
		return failedResult(result, fmt.Errorf("failed to resolve version: %w", err))
	}
	result.PluginID = resolvedID

	metadata, err := m.getPackageMetadata(ctx, resolvedID)
	if err != nil {
		return failedResult(result, fmt.Errorf("failed to get package metadata: %w", err))
	}

	if err := m.validatePackageMetadata(metadata); err != nil {
		return failedResult(result, err)
	}

	archivePath, cleanup, err := m.downloadAndVerifyPlugin(ctx, resolvedID, metadata, progressFn)
	if err != nil {
		return failedResult(result, err)
	}
	defer cleanup()

	// Stage: Extracting
	reportProgress(progressFn, resolvedID, StageExtracting, 0, 0)

	if err := m.extractAndInstallPlugin(resolvedID, archivePath, metadata); err != nil {
		return failedResult(result, err)
	}

	reportProgress(progressFn, resolvedID, StageComplete, 0, 0)
	result.Status = StatusInstalled
	return result
}

func failedResult(result *InstallResult, err error) *InstallResult {
	result.Status = StatusFailed
	result.Error = err
	return result
}

func reportProgress(fn ProgressCallback, id *PluginID, stage InstallStage, downloaded, total int64) {
	if fn != nil {
		fn(id, stage, downloaded, total)
	}
}

func (m *Manager) checkIfAlreadyInstalled(pluginID *PluginID) (*InstallResult, error) {
	installed, existingPlugin, err := m.IsInstalled(pluginID)
	if err != nil {
		return nil, err
	}
	if installed && existingPlugin != nil {
		return &InstallResult{PluginID: pluginID, Status: StatusSkipped}, nil
	}
	return nil, nil
}

func (m *Manager) resolvePluginVersion(ctx context.Context, pluginID *PluginID) (*PluginID, error) {
	resolvedVersion, err := m.ResolveVersion(ctx, pluginID)
	if err != nil {
		return nil, err
	}
	return pluginID.WithVersion(resolvedVersion), nil
}

func (m *Manager) getPackageMetadata(
	ctx context.Context,
	pluginID *PluginID,
) (*registries.PluginPackageMetadata, error) {
	return m.registryClient.GetPackageMetadata(
		ctx,
		pluginID.RegistryHost,
		pluginID.Namespace,
		pluginID.Name,
		pluginID.Version,
		runtime.GOOS,
		runtime.GOARCH,
	)
}

func (m *Manager) validatePackageMetadata(metadata *registries.PluginPackageMetadata) error {
	if metadata.ShasumsURL == "" || metadata.ShasumsSignatureURL == "" {
		return registries.ErrSignatureMissing
	}
	if len(metadata.SigningKeys) == 0 {
		return registries.ErrSigningKeysMissing
	}
	return nil
}

func (m *Manager) downloadAndVerifyPlugin(
	ctx context.Context,
	pluginID *PluginID,
	metadata *registries.PluginPackageMetadata,
	progressFn ProgressCallback,
) (archivePath string, cleanup func(), err error) {
	tempDir, err := os.MkdirTemp("", "bluelink-plugin-*")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	cleanup = func() { os.RemoveAll(tempDir) }

	archivePath = filepath.Join(tempDir, metadata.Filename)

	// Stage: Downloading
	reportProgress(progressFn, pluginID, StageDownloading, 0, 0)

	err = m.registryClient.DownloadPackage(
		ctx,
		pluginID.RegistryHost,
		metadata,
		archivePath,
		func(downloaded, total int64) {
			reportProgress(progressFn, pluginID, StageDownloading, downloaded, total)
		},
	)
	if err != nil {
		cleanup()
		return "", nil, fmt.Errorf("failed to download plugin: %w", err)
	}

	// Stage: Verifying
	reportProgress(progressFn, pluginID, StageVerifying, 0, 0)

	if err := m.verifyPluginSignatureAndChecksum(ctx, pluginID, metadata, archivePath); err != nil {
		cleanup()
		return "", nil, err
	}

	return archivePath, cleanup, nil
}

func (m *Manager) verifyPluginSignatureAndChecksum(
	ctx context.Context,
	pluginID *PluginID,
	metadata *registries.PluginPackageMetadata,
	archivePath string,
) error {
	shasums, err := m.registryClient.DownloadShasums(ctx, pluginID.RegistryHost, metadata.ShasumsURL)
	if err != nil {
		return fmt.Errorf("failed to download checksums: %w", err)
	}

	signature, err := m.registryClient.DownloadSignature(
		ctx, pluginID.RegistryHost, metadata.ShasumsSignatureURL,
	)
	if err != nil {
		return fmt.Errorf("failed to download signature: %w", err)
	}

	if err := m.VerifyGPGSignature(shasums, signature, metadata.SigningKeys); err != nil {
		return fmt.Errorf("%w: %v", registries.ErrSignatureInvalid, err)
	}

	if err := m.VerifyChecksum(archivePath, shasums, metadata.Filename); err != nil {
		return fmt.Errorf("%w: %v", registries.ErrChecksumMismatch, err)
	}

	return nil
}

func (m *Manager) extractAndInstallPlugin(
	pluginID *PluginID,
	archivePath string,
	metadata *registries.PluginPackageMetadata,
) error {
	// Plugin executables are extracted to the bin subdirectory
	destDir := filepath.Join(m.pluginsDir, "bin", pluginID.Namespace, pluginID.Name, pluginID.Version)
	if err := os.MkdirAll(destDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugin directory: %w", err)
	}

	if err := m.ExtractArchive(archivePath, destDir); err != nil {
		return fmt.Errorf("%w: %v", registries.ErrExtractionFailed, err)
	}

	if err := m.addToManifest(pluginID, metadata.Shasum); err != nil {
		return fmt.Errorf("failed to update manifest: %w", err)
	}

	return nil
}

// ResolveLatestVersion finds the latest version of a plugin.
func (m *Manager) ResolveLatestVersion(ctx context.Context, pluginID *PluginID) (string, error) {
	versions, err := m.registryClient.ListVersions(
		ctx,
		pluginID.RegistryHost,
		pluginID.Namespace,
		pluginID.Name,
	)
	if err != nil {
		return "", err
	}

	if len(versions.Versions) == 0 {
		return "", registries.ErrVersionNotFound
	}

	// Return the first version (registry should return sorted, latest first)
	return versions.Versions[0].Version, nil
}

// ResolveVersion resolves the version for a plugin.
// If no version is specified, returns the latest version.
// If a constraint is specified (^1.0.0 or ~1.0.0), returns the best matching version.
// If an exact version is specified, returns that version.
func (m *Manager) ResolveVersion(ctx context.Context, pluginID *PluginID) (string, error) {
	// No version specified - get latest
	if pluginID.Version == "" {
		return m.ResolveLatestVersion(ctx, pluginID)
	}

	// Parse the constraint
	constraint, err := pluginID.ParseVersionConstraint()
	if err != nil {
		return "", fmt.Errorf("invalid version constraint %q: %w", pluginID.Version, err)
	}

	// Exact version - return as-is
	if constraint.IsExact() {
		return pluginID.Version, nil
	}

	// Constraint - find best match from available versions
	return m.resolveBestMatchingVersion(ctx, pluginID, constraint)
}

func (m *Manager) resolveBestMatchingVersion(
	ctx context.Context,
	pluginID *PluginID,
	constraint *version.Constraint,
) (string, error) {
	resp, err := m.registryClient.ListVersions(
		ctx,
		pluginID.RegistryHost,
		pluginID.Namespace,
		pluginID.Name,
	)
	if err != nil {
		return "", err
	}

	// Parse all available versions
	var versions []*version.Version
	for _, vi := range resp.Versions {
		v, err := version.Parse(vi.Version)
		if err != nil {
			continue // Skip unparseable versions
		}
		versions = append(versions, v)
	}

	// Find best match
	best := constraint.FindBestMatch(versions)
	if best == nil {
		return "", fmt.Errorf("no version matching constraint %s found for %s",
			constraint.String(), pluginID.String())
	}

	return best.String(), nil
}

// resolveDependencies resolves all dependencies for the given plugins and returns
// them in topological order (dependencies first).
func (m *Manager) resolveDependencies(
	ctx context.Context,
	pluginIDs []*PluginID,
) ([]*PluginID, error) {
	resolver := newDependencyResolver(m, ctx)
	return resolver.resolveAll(pluginIDs)
}

// dependencyResolver handles dependency resolution with cycle detection.
type dependencyResolver struct {
	manager *Manager
	ctx     context.Context
	visited map[string]*PluginID
	inStack map[string]bool
	result  []*PluginID
}

func newDependencyResolver(m *Manager, ctx context.Context) *dependencyResolver {
	return &dependencyResolver{
		manager: m,
		ctx:     ctx,
		visited: make(map[string]*PluginID),
		inStack: make(map[string]bool),
	}
}

func (r *dependencyResolver) resolveAll(pluginIDs []*PluginID) ([]*PluginID, error) {
	for _, pluginID := range pluginIDs {
		if err := r.resolve(pluginID); err != nil {
			return nil, err
		}
	}
	return r.result, nil
}

func (r *dependencyResolver) resolve(pluginID *PluginID) error {
	key := pluginID.FullyQualified()

	if r.inStack[key] {
		return fmt.Errorf("circular dependency detected involving %s", key)
	}
	if _, exists := r.visited[key]; exists {
		return nil
	}

	r.inStack[key] = true
	defer func() { r.inStack[key] = false }()

	resolvedID, err := r.resolveVersion(pluginID)
	if err != nil {
		return err
	}

	if installed, _, _ := r.manager.IsInstalled(resolvedID); installed {
		r.visited[key] = resolvedID
		return nil
	}

	if err := r.resolvePluginDependencies(resolvedID); err != nil {
		return err
	}

	r.visited[key] = resolvedID
	r.result = append(r.result, resolvedID)
	return nil
}

func (r *dependencyResolver) resolveVersion(pluginID *PluginID) (*PluginID, error) {
	resolvedVersion, err := r.manager.ResolveVersion(r.ctx, pluginID)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve version for %s: %w", pluginID.FullyQualified(), err)
	}
	return pluginID.WithVersion(resolvedVersion), nil
}

func (r *dependencyResolver) resolvePluginDependencies(resolvedID *PluginID) error {
	metadata, err := r.manager.registryClient.GetPackageMetadata(
		r.ctx,
		resolvedID.RegistryHost,
		resolvedID.Namespace,
		resolvedID.Name,
		resolvedID.Version,
		runtime.GOOS,
		runtime.GOARCH,
	)
	if err != nil {
		return fmt.Errorf("failed to get metadata for %s: %w", resolvedID.FullyQualified(), err)
	}

	for depIDStr, depVersion := range metadata.Dependencies {
		depID, err := r.buildDependencyID(depIDStr, depVersion, resolvedID)
		if err != nil {
			return err
		}
		if err := r.resolve(depID); err != nil {
			return err
		}
	}
	return nil
}

func (r *dependencyResolver) buildDependencyID(
	depIDStr, depVersion string,
	parent *PluginID,
) (*PluginID, error) {
	depID, err := ParsePluginID(depIDStr)
	if err != nil {
		return nil, fmt.Errorf("invalid dependency %s for %s: %w", depIDStr, parent.FullyQualified(), err)
	}

	if depID.Version == "" && depVersion != "" {
		depID = depID.WithVersion(depVersion)
	}

	// Inherit registry from parent if dependency doesn't specify one
	if depID.IsDefaultRegistry() && !parent.IsDefaultRegistry() {
		depID = &PluginID{
			RegistryHost: parent.RegistryHost,
			Namespace:    depID.Namespace,
			Name:         depID.Name,
			Version:      depID.Version,
		}
	}
	return depID, nil
}

// VerifyGPGSignature verifies the GPG signature of the shasums file.
func (m *Manager) VerifyGPGSignature(shasums, signature []byte, signingKeys map[string]string) error {
	if len(signingKeys) == 0 {
		return fmt.Errorf("no signing keys provided")
	}

	// Build keyring from provided keys
	var keyring openpgp.EntityList
	for keyID, keyData := range signingKeys {
		entities, err := openpgp.ReadArmoredKeyRing(strings.NewReader(keyData))
		if err != nil {
			return fmt.Errorf("failed to parse signing key %s: %w", keyID, err)
		}
		keyring = append(keyring, entities...)
	}

	if len(keyring) == 0 {
		return fmt.Errorf("no valid signing keys found")
	}

	// Verify the signature
	_, err := openpgp.CheckArmoredDetachedSignature(
		keyring,
		strings.NewReader(string(shasums)),
		strings.NewReader(string(signature)),
		nil,
	)
	if err != nil {
		return fmt.Errorf("signature verification failed: %w", err)
	}

	return nil
}

// VerifyChecksum verifies the SHA256 checksum of a file against the shasums content.
func (m *Manager) VerifyChecksum(filePath string, shasums []byte, expectedFilename string) error {
	// Parse shasums to find expected checksum
	expectedChecksum, err := m.extractChecksum(shasums, expectedFilename)
	if err != nil {
		return err
	}

	// Calculate actual checksum
	actualChecksum, err := m.calculateSHA256(filePath)
	if err != nil {
		return err
	}

	if actualChecksum != expectedChecksum {
		return fmt.Errorf("checksum mismatch: expected %s, got %s", expectedChecksum, actualChecksum)
	}

	return nil
}

func (m *Manager) extractChecksum(shasums []byte, filename string) (string, error) {
	lines := strings.SplitSeq(string(shasums), "\n")
	for line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Format: "checksum  filename" or "checksum filename"
		parts := strings.Fields(line)
		if len(parts) >= 2 {
			checksum := parts[0]
			name := parts[len(parts)-1]
			if name == filename || strings.HasSuffix(name, "/"+filename) {
				return strings.ToLower(checksum), nil
			}
		}
	}

	return "", fmt.Errorf("checksum not found for %s in shasums file", filename)
}

func (m *Manager) calculateSHA256(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := sha256.New()
	if _, err := io.Copy(hash, file); err != nil {
		return "", err
	}

	return hex.EncodeToString(hash.Sum(nil)), nil
}

// ExtractArchive extracts a tar.gz archive to the destination directory.
func (m *Manager) ExtractArchive(archivePath, destDir string) error {
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if err := extractTarEntry(tr, header, destDir); err != nil {
			return err
		}
	}
	return nil
}

func extractTarEntry(tr *tar.Reader, header *tar.Header, destDir string) error {
	targetPath, err := validateEntryPath(destDir, header.Name)
	if err != nil {
		return err
	}

	switch header.Typeflag {
	case tar.TypeDir:
		return os.MkdirAll(targetPath, 0755)
	case tar.TypeReg:
		return extractRegularFile(tr, targetPath, os.FileMode(header.Mode))
	case tar.TypeSymlink:
		return createSafeSymlink(destDir, targetPath, header.Linkname)
	default:
		return nil // Skip unknown types
	}
}

// createSafeSymlink creates a symlink after validating the target stays within destDir.
// This prevents zip slip attacks via symlinks that point outside the extraction directory.
func createSafeSymlink(destDir, symlinkPath, linkTarget string) error {
	// Resolve the symlink target relative to the symlink's directory
	symlinkDir := filepath.Dir(symlinkPath)
	resolvedTarget := filepath.Clean(filepath.Join(symlinkDir, linkTarget))

	// Use filepath.Rel to verify the resolved target is within destDir
	// This is the pattern recognized by security scanners for zip slip prevention
	relPath, err := filepath.Rel(destDir, resolvedTarget)
	if err != nil {
		return fmt.Errorf("symlink target escapes destination: %s -> %s", symlinkPath, linkTarget)
	}

	// Reject if the relative path escapes the destination (starts with "..")
	if strings.HasPrefix(relPath, "..") {
		return fmt.Errorf("symlink target escapes destination: %s -> %s", symlinkPath, linkTarget)
	}

	return os.Symlink(linkTarget, symlinkPath)
}

func validateEntryPath(destDir, entryName string) (string, error) {
	targetPath := filepath.Join(destDir, entryName)
	cleanDest := filepath.Clean(destDir) + string(os.PathSeparator)
	if !strings.HasPrefix(filepath.Clean(targetPath), cleanDest) {
		return "", fmt.Errorf("invalid file path: %s", entryName)
	}
	return targetPath, nil
}

func extractRegularFile(tr *tar.Reader, targetPath string, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(targetPath), 0755); err != nil {
		return err
	}

	outFile, err := os.OpenFile(targetPath, os.O_CREATE|os.O_RDWR|os.O_TRUNC, mode)
	if err != nil {
		return err
	}
	defer outFile.Close()

	_, err = io.Copy(outFile, tr)
	return err
}

// LoadManifest loads the plugin manifest from disk.
func (m *Manager) LoadManifest() (*PluginManifest, error) {
	manifestPath := filepath.Join(m.pluginsDir, "manifest.json")

	data, err := os.ReadFile(manifestPath)
	if os.IsNotExist(err) {
		return &PluginManifest{Plugins: make(map[string]*InstalledPlugin)}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read manifest: %w", err)
	}

	var manifest PluginManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("failed to parse manifest: %w", err)
	}

	if manifest.Plugins == nil {
		manifest.Plugins = make(map[string]*InstalledPlugin)
	}

	return &manifest, nil
}

// SaveManifest saves the plugin manifest to disk.
func (m *Manager) SaveManifest(manifest *PluginManifest) error {
	if err := os.MkdirAll(m.pluginsDir, 0755); err != nil {
		return fmt.Errorf("failed to create plugins directory: %w", err)
	}

	manifestPath := filepath.Join(m.pluginsDir, "manifest.json")

	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write manifest: %w", err)
	}

	return nil
}

func (m *Manager) addToManifest(pluginID *PluginID, shasum string) error {
	manifest, err := m.LoadManifest()
	if err != nil {
		return err
	}

	key := pluginID.ManifestKey()
	manifest.Plugins[key] = &InstalledPlugin{
		ID:           pluginID.String(),
		Version:      pluginID.Version,
		RegistryHost: pluginID.RegistryHost,
		Shasum:       shasum,
		InstalledAt:  time.Now(),
	}

	return m.SaveManifest(manifest)
}

// ListInstalled returns all installed plugins.
func (m *Manager) ListInstalled() ([]*InstalledPlugin, error) {
	manifest, err := m.LoadManifest()
	if err != nil {
		return nil, err
	}

	plugins := make([]*InstalledPlugin, 0, len(manifest.Plugins))
	for _, plugin := range manifest.Plugins {
		plugins = append(plugins, plugin)
	}

	// Sort by ID for consistent output
	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].ID < plugins[j].ID
	})

	return plugins, nil
}
