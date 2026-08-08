package deployconfig

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
)

type ResolverSuite struct {
	suite.Suite
	logger *zap.Logger
}

func (s *ResolverSuite) SetupSuite() {
	s.logger = zap.NewNop()
}

func (s *ResolverSuite) resolver() *Resolver {
	return NewResolver(ResolverConfig{}, s.logger)
}

func (s *ResolverSuite) fixtureDir(name string) string {
	path, err := filepath.Abs(filepath.Join("__testdata", name))
	s.Require().NoError(err)
	return path
}

func (s *ResolverSuite) contextVariable(params core.BlueprintParams, name string) string {
	value := params.ContextVariable(name)
	s.Require().NotNil(value, "expected context variable %q to be set", name)
	s.Require().NotNil(value.StringValue)
	return *value.StringValue
}

func (s *ResolverSuite) TestResolve_FindsAppConfigInSameDirectory() {
	result := s.resolver().Resolve(s.fixtureDir("app-config"))

	s.True(result.Usable())
	s.NoError(result.Err)
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))
	s.Equal("walk-app", s.contextVariable(result.Params(), "celerity.appName"))
}

func (s *ResolverSuite) TestResolve_WalksUpToParentDirectories() {
	result := s.resolver().Resolve(s.fixtureDir(filepath.Join("app-config", "nested", "deeper")))

	s.True(result.Usable())
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))
}

func (s *ResolverSuite) TestResolve_SplitsProviderAndTransformerConfig() {
	result := s.resolver().Resolve(s.fixtureDir("app-config"))
	params := result.Params()

	// Provider keys have the namespace prefix stripped.
	region := params.ProviderConfig("aws")["region"]
	s.Require().NotNil(region)
	s.Equal("us-east-1", *region.StringValue)

	// Transformer keys keep their full dotted form.
	retention := params.TransformerConfig("celerity")["aws.sqs.messageRetentionPeriod"]
	s.Require().NotNil(retention)
	s.Require().NotNil(retention.IntValue)
	s.Equal(60, *retention.IntValue)

	// A transformer-prefixed key must not leak into provider config.
	s.Nil(params.ProviderConfig("aws")["sqs.messageRetentionPeriod"])
}

func (s *ResolverSuite) TestResolve_PrefersAppConfigOverGeneratedConfig() {
	result := s.resolver().Resolve(s.fixtureDir("both"))

	s.True(result.Usable())
	s.Equal("app.deploy.jsonc", filepath.Base(result.SourcePath))
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))
	s.Equal("source-of-truth-app", s.contextVariable(result.Params(), "celerity.appName"))
}

func (s *ResolverSuite) TestResolve_FallsBackToGeneratedConfig() {
	result := s.resolver().Resolve(s.fixtureDir("generated-only"))

	s.True(result.Usable())
	s.Equal("deploy-config.json", filepath.Base(result.SourcePath))
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))
}

func (s *ResolverSuite) TestResolve_ReportsUnsupportedDeployTarget() {
	result := s.resolver().Resolve(s.fixtureDir("unsupported-target"))

	s.False(result.Usable())
	s.Require().Error(result.Err)
	s.Contains(result.Err.Error(), "not-a-real-target")
	s.NotEmpty(result.SourcePath)
}

func (s *ResolverSuite) TestResolve_ReportsMalformedConfig() {
	result := s.resolver().Resolve(s.fixtureDir("malformed"))

	s.False(result.Usable())
	s.Require().Error(result.Err)
	s.NotEmpty(result.SourcePath)
}

func (s *ResolverSuite) TestResolve_UnusableConfigFallsBackToValidationParams() {
	result := s.resolver().Resolve(s.fixtureDir("unsupported-target"))
	params := result.Params()

	s.Empty(params.ProviderConfig("aws"))
	s.Nil(params.ContextVariable("deployTarget"))
	s.Require().NotNil(params.ContextVariable(core.ValidationContextVariableName))
}

func (s *ResolverSuite) TestResolve_NoConfigIsNotAnError() {
	dir := s.T().TempDir()

	result := s.resolver().Resolve(dir)

	s.False(result.Usable())
	s.NoError(result.Err)
	s.Empty(result.SourcePath)
	s.Require().NotNil(result.Params().ContextVariable(core.ValidationContextVariableName))
}

// bluelink.deploy.jsonc in the project root is the documented convention for a
// Bluelink project and needs no conversion.
func (s *ResolverSuite) TestResolve_FindsCanonicalBluelinkConfig() {
	result := s.resolver().Resolve(s.fixtureDir("bluelink-config"))

	s.True(result.Usable())
	s.Equal(BluelinkDeployConfigFile, filepath.Base(result.SourcePath))
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))

	region := result.Params().ProviderConfig("aws")["region"]
	s.Require().NotNil(region)
	s.Equal("eu-west-2", *region.StringValue)
}

func (s *ResolverSuite) TestResolve_FindsCanonicalBluelinkConfigFromNestedDirectory() {
	result := s.resolver().Resolve(s.fixtureDir(filepath.Join("bluelink-config", "nested")))

	s.True(result.Usable())
	s.Equal(BluelinkDeployConfigFile, filepath.Base(result.SourcePath))
}

func (s *ResolverSuite) TestResolve_PrefersCanonicalConfigOverCelerityAuthoringConfig() {
	result := s.resolver().Resolve(s.fixtureDir("both-conventions"))

	s.True(result.Usable())
	s.Equal(BluelinkDeployConfigFile, filepath.Base(result.SourcePath))
	s.Equal("aws", s.contextVariable(result.Params(), "deployTarget"))
}

// A template holds placeholders rather than real values, so discovering one
// would feed nonsense configuration into validation.
func (s *ResolverSuite) TestResolve_IgnoresTemplateFiles() {
	result := s.resolver().Resolve(s.fixtureDir("template-only"))

	s.False(result.Usable())
	s.NoError(result.Err)
	s.Empty(result.SourcePath)
}

// Which environment's file applies is a deliberate choice, so a per-environment
// file is named explicitly rather than discovered. A relative name has to
// resolve against the project, not the server's working directory.
func (s *ResolverSuite) TestResolve_UsesRelativeExplicitPathPerProject() {
	resolver := NewResolver(
		ResolverConfig{ExplicitPath: "bluelink.deploy.prod.jsonc"},
		s.logger,
	)

	result := resolver.Resolve(s.fixtureDir("env-variants"))

	s.True(result.Usable())
	s.Equal("bluelink.deploy.prod.jsonc", filepath.Base(result.SourcePath))
	s.Equal("aws", s.contextVariable(result.Params(), "deployTarget"))
	s.Equal("prod", s.contextVariable(result.Params(), "appEnv"))
}

// A per-environment variant of Celerity's authoring file still needs converting,
// which follows from the name rather than an exact match.
func (s *ResolverSuite) TestResolve_ConvertsCelerityAuthoringVariant() {
	resolver := NewResolver(
		ResolverConfig{ExplicitPath: "app.deploy.staging.jsonc"},
		s.logger,
	)

	result := resolver.Resolve(s.fixtureDir("celerity-env-variant"))

	s.True(result.Usable())
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))
	s.Equal("staging-app", s.contextVariable(result.Params(), "celerity.appName"))
	s.Equal("staging", s.contextVariable(result.Params(), "appEnv"))
}

// A project keeping deploy configuration under its own names needs to say so
// rather than have the conventions widened here.
func (s *ResolverSuite) TestResolve_UsesConfiguredCandidatePaths() {
	dir := s.T().TempDir()
	s.writeFile(
		filepath.Join(dir, "custom.deploy.jsonc"),
		`{"contextVariables":{"deployTarget":"azure"}}`,
	)

	resolver := NewResolver(
		ResolverConfig{CandidatePaths: []string{"custom.deploy.jsonc"}},
		s.logger,
	)
	result := resolver.Resolve(dir)

	s.True(result.Usable())
	s.Equal("custom.deploy.jsonc", filepath.Base(result.SourcePath))
	s.Equal("azure", s.contextVariable(result.Params(), "deployTarget"))
}

// Configured names replace the conventions rather than adding to them, so a
// project naming its configuration explicitly cannot silently pick up a
// conventionally named file sitting alongside it.
func (s *ResolverSuite) TestResolve_ConfiguredCandidatePathsReplaceConventions() {
	dir := s.T().TempDir()
	s.writeFile(
		filepath.Join(dir, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"aws"}}`,
	)

	resolver := NewResolver(
		ResolverConfig{CandidatePaths: []string{"custom.deploy.jsonc"}},
		s.logger,
	)
	result := resolver.Resolve(dir)

	s.False(result.Usable())
	s.Empty(result.SourcePath)
}

// Configuration above the workspace belongs to something else. Picking it up
// would apply a deploy target the user never chose for this project, and would
// do so silently.
func (s *ResolverSuite) TestResolve_StopsAtWorkspaceRoot() {
	outside := s.T().TempDir()
	root := filepath.Join(outside, "workspace")
	nested := filepath.Join(root, "services", "orders")
	s.Require().NoError(os.MkdirAll(nested, 0755))

	// Configuration one level above the workspace root.
	s.writeFile(
		filepath.Join(outside, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"aws"}}`,
	)

	resolver := NewResolver(ResolverConfig{WorkspaceRoots: []string{root}}, s.logger)
	result := resolver.Resolve(nested)

	s.False(result.Usable())
	s.Empty(result.SourcePath)
}

func (s *ResolverSuite) TestResolve_SearchesTheWorkspaceRootItself() {
	root := s.T().TempDir()
	nested := filepath.Join(root, "services", "orders")
	s.Require().NoError(os.MkdirAll(nested, 0755))
	s.writeFile(
		filepath.Join(root, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"aws-serverless"}}`,
	)

	resolver := NewResolver(ResolverConfig{WorkspaceRoots: []string{root}}, s.logger)
	result := resolver.Resolve(nested)

	s.True(result.Usable())
	s.Equal("aws-serverless", s.contextVariable(result.Params(), "deployTarget"))
}

// A client may send no workspace folders, so the repository root stands in. This
// keeps a blueprint opened as a loose file from reaching a home directory.
func (s *ResolverSuite) TestResolve_StopsAtRepositoryRootWhenNoWorkspaceRoot() {
	outside := s.T().TempDir()
	repo := filepath.Join(outside, "repo")
	nested := filepath.Join(repo, "services", "orders")
	s.Require().NoError(os.MkdirAll(nested, 0755))
	s.Require().NoError(os.MkdirAll(filepath.Join(repo, repositoryMarker), 0755))

	s.writeFile(
		filepath.Join(outside, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"aws"}}`,
	)

	result := s.resolver().Resolve(nested)

	s.False(result.Usable())
	s.Empty(result.SourcePath)
}

func (s *ResolverSuite) TestResolve_SearchesTheRepositoryRootItself() {
	repo := s.T().TempDir()
	nested := filepath.Join(repo, "services", "orders")
	s.Require().NoError(os.MkdirAll(nested, 0755))
	s.Require().NoError(os.MkdirAll(filepath.Join(repo, repositoryMarker), 0755))
	s.writeFile(
		filepath.Join(repo, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"azure"}}`,
	)

	result := s.resolver().Resolve(nested)

	s.True(result.Usable())
	s.Equal("azure", s.contextVariable(result.Params(), "deployTarget"))
}

// Several folders open at once each bound their own subtree.
func (s *ResolverSuite) TestResolve_HonoursMultipleWorkspaceRoots() {
	parent := s.T().TempDir()
	first := filepath.Join(parent, "first")
	second := filepath.Join(parent, "second")
	s.Require().NoError(os.MkdirAll(filepath.Join(first, "nested"), 0755))
	s.Require().NoError(os.MkdirAll(second, 0755))

	s.writeFile(
		filepath.Join(first, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"aws-serverless"}}`,
	)
	s.writeFile(
		filepath.Join(parent, BluelinkDeployConfigFile),
		`{"contextVariables":{"deployTarget":"gcloud"}}`,
	)

	resolver := NewResolver(
		ResolverConfig{WorkspaceRoots: []string{first, second}},
		s.logger,
	)

	// Resolves within its own root.
	s.Equal(
		"aws-serverless",
		s.contextVariable(resolver.Resolve(filepath.Join(first, "nested")).Params(), "deployTarget"),
	)

	// The other root has none of its own and must not reach the shared parent.
	s.False(resolver.Resolve(second).Usable())
}

func (s *ResolverSuite) TestResolve_UsesExplicitPath() {
	explicit := filepath.Join(s.fixtureDir("explicit"), "custom-deploy-config.json")
	resolver := NewResolver(ResolverConfig{ExplicitPath: explicit}, s.logger)

	// A directory with its own configuration must still yield the explicit one.
	result := resolver.Resolve(s.fixtureDir("app-config"))

	s.True(result.Usable())
	s.Equal(explicit, result.SourcePath)
	s.Equal("azure-serverless", s.contextVariable(result.Params(), "deployTarget"))
}

// The validation flag tells plugins they are running in an editor rather than
// a deployment, so a deploy configuration must never be able to unset it.
func (s *ResolverSuite) TestResolve_ValidationContextFlagOverridesConfig() {
	dir := s.T().TempDir()
	s.writeFile(filepath.Join(dir, "app.deploy.jsonc"), `{
		"appName": "flag-app",
		"deployTarget": { "name": "aws-serverless" },
		"contextVariables": { "`+core.ValidationContextVariableName+`": false }
	}`)

	params := s.resolver().Resolve(dir).Params()

	flag := params.ContextVariable(core.ValidationContextVariableName)
	s.Require().NotNil(flag)
	s.Require().NotNil(flag.BoolValue)
	s.True(*flag.BoolValue)
}

func (s *ResolverSuite) TestResolve_PicksUpEditsToConfig() {
	dir := s.T().TempDir()
	configPath := filepath.Join(dir, "app.deploy.jsonc")
	resolver := s.resolver()

	s.writeFile(configPath, `{"appName":"a","deployTarget":{"name":"aws-serverless"}}`)
	s.Equal("aws-serverless", s.contextVariable(resolver.Resolve(dir).Params(), "deployTarget"))

	s.writeFile(configPath, `{"appName":"a","deployTarget":{"name":"azure"}}`)
	s.Equal("azure", s.contextVariable(resolver.Resolve(dir).Params(), "deployTarget"))
}

func (s *ResolverSuite) TestResolve_IgnoresNonScalarConfigValues() {
	dir := s.T().TempDir()
	s.writeFile(filepath.Join(dir, "app.deploy.jsonc"), `{
		"appName": "nested-app",
		"deployTarget": {
			"name": "aws-serverless",
			"config": { "aws.region": "us-east-1", "aws.nested": { "key": "value" } }
		}
	}`)

	params := s.resolver().Resolve(dir).Params()

	s.Require().NotNil(params.ProviderConfig("aws")["region"])
	s.Nil(params.ProviderConfig("aws")["nested"])
}

// TestConversionMatchesCLIOutput pins the conversion against a deploy config
// file generated by the Celerity CLI from the paired input. Regenerate both
// files together when the CLI's conversion changes.
func (s *ResolverSuite) TestConversionMatchesCLIOutput() {
	goldenDir := s.fixtureDir("golden")

	source := &CelerityAppSource{}
	converted, err := source.Load(filepath.Join(goldenDir, "app.deploy.jsonc"))
	s.Require().NoError(err)

	expectedData, err := os.ReadFile(filepath.Join(goldenDir, "deploy-config.json"))
	s.Require().NoError(err)
	expected := &Config{}
	s.Require().NoError(json.Unmarshal(expectedData, expected))

	s.Equal(expected.Providers, converted.Providers)
	s.Equal(expected.Transformers, converted.Transformers)
	s.Equal(expected.ContextVariables, converted.ContextVariables)
	s.Equal(expected.BlueprintVariables, converted.BlueprintVariables)
}

func (s *ResolverSuite) writeFile(path string, content string) {
	s.Require().NoError(os.WriteFile(path, []byte(content), 0644))
}

func TestResolverSuite(t *testing.T) {
	suite.Run(t, new(ResolverSuite))
}
