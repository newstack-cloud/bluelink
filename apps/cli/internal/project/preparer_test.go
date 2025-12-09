package project

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type PreparerSuite struct {
	suite.Suite
	tempDir  string
	preparer Preparer
}

func (s *PreparerSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "preparer-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir
	s.preparer = NewDefaultPreparer()
}

func (s *PreparerSuite) TearDownTest() {
	os.RemoveAll(s.tempDir)
}

// RemoveGitHistory tests

func (s *PreparerSuite) Test_RemoveGitHistory_removes_git_directory() {
	// Create a .git directory
	gitDir := filepath.Join(s.tempDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	s.Require().NoError(err)

	// Create a file inside .git
	err = os.WriteFile(filepath.Join(gitDir, "config"), []byte("test"), 0644)
	s.Require().NoError(err)

	// Remove git history
	err = s.preparer.RemoveGitHistory(s.tempDir)
	s.NoError(err)

	// Verify .git is removed
	_, err = os.Stat(gitDir)
	s.True(os.IsNotExist(err))
}

func (s *PreparerSuite) Test_RemoveGitHistory_succeeds_when_no_git_directory() {
	// No .git directory exists
	err := s.preparer.RemoveGitHistory(s.tempDir)
	s.NoError(err)
}

// RemoveMaintainerFiles tests

func (s *PreparerSuite) Test_RemoveMaintainerFiles_removes_readme_and_license() {
	// Create maintainer files
	readmePath := filepath.Join(s.tempDir, "README.md")
	licensePath := filepath.Join(s.tempDir, "LICENSE")

	err := os.WriteFile(readmePath, []byte("# Template README"), 0644)
	s.Require().NoError(err)
	err = os.WriteFile(licensePath, []byte("MIT License"), 0644)
	s.Require().NoError(err)

	// Remove maintainer files
	err = s.preparer.RemoveMaintainerFiles(s.tempDir)
	s.NoError(err)

	// Verify files are removed
	_, err = os.Stat(readmePath)
	s.True(os.IsNotExist(err))
	_, err = os.Stat(licensePath)
	s.True(os.IsNotExist(err))
}

func (s *PreparerSuite) Test_RemoveMaintainerFiles_succeeds_when_files_missing() {
	// No maintainer files exist
	err := s.preparer.RemoveMaintainerFiles(s.tempDir)
	s.NoError(err)
}

func (s *PreparerSuite) Test_RemoveMaintainerFiles_removes_only_existing_files() {
	// Create only README.md
	readmePath := filepath.Join(s.tempDir, "README.md")
	err := os.WriteFile(readmePath, []byte("# Template README"), 0644)
	s.Require().NoError(err)

	// Remove maintainer files
	err = s.preparer.RemoveMaintainerFiles(s.tempDir)
	s.NoError(err)

	// Verify README is removed
	_, err = os.Stat(readmePath)
	s.True(os.IsNotExist(err))
}

// SelectBlueprintFormat tests

func (s *PreparerSuite) Test_SelectBlueprintFormat_yaml_removes_jsonc_file() {
	// Create both blueprint template files
	yamlPath := filepath.Join(s.tempDir, "project.blueprint.yaml.tmpl")
	jsoncPath := filepath.Join(s.tempDir, "project.blueprint.jsonc.tmpl")

	err := os.WriteFile(yamlPath, []byte("version: 1.0"), 0644)
	s.Require().NoError(err)
	err = os.WriteFile(jsoncPath, []byte(`{"version": "1.0"}`), 0644)
	s.Require().NoError(err)

	// Select yaml format
	err = s.preparer.SelectBlueprintFormat(s.tempDir, "yaml")
	s.NoError(err)

	// Verify yaml file still exists
	_, err = os.Stat(yamlPath)
	s.NoError(err)

	// Verify jsonc file is removed
	_, err = os.Stat(jsoncPath)
	s.True(os.IsNotExist(err))
}

func (s *PreparerSuite) Test_SelectBlueprintFormat_jsonc_removes_yaml_file() {
	// Create both blueprint template files
	yamlPath := filepath.Join(s.tempDir, "project.blueprint.yaml.tmpl")
	jsoncPath := filepath.Join(s.tempDir, "project.blueprint.jsonc.tmpl")

	err := os.WriteFile(yamlPath, []byte("version: 1.0"), 0644)
	s.Require().NoError(err)
	err = os.WriteFile(jsoncPath, []byte(`{"version": "1.0"}`), 0644)
	s.Require().NoError(err)

	// Select jsonc format
	err = s.preparer.SelectBlueprintFormat(s.tempDir, "jsonc")
	s.NoError(err)

	// Verify jsonc file still exists
	_, err = os.Stat(jsoncPath)
	s.NoError(err)

	// Verify yaml file is removed
	_, err = os.Stat(yamlPath)
	s.True(os.IsNotExist(err))
}

func (s *PreparerSuite) Test_SelectBlueprintFormat_returns_error_for_unsupported_format() {
	err := s.preparer.SelectBlueprintFormat(s.tempDir, "xml")
	s.Error(err)
	s.Contains(err.Error(), "unsupported blueprint format")
}

func (s *PreparerSuite) Test_SelectBlueprintFormat_returns_error_when_selected_file_missing() {
	// Create only jsonc file
	jsoncPath := filepath.Join(s.tempDir, "project.blueprint.jsonc.tmpl")
	err := os.WriteFile(jsoncPath, []byte(`{"version": "1.0"}`), 0644)
	s.Require().NoError(err)

	// Try to select yaml format (which doesn't exist)
	err = s.preparer.SelectBlueprintFormat(s.tempDir, "yaml")
	s.Error(err)
	s.Contains(err.Error(), "not found")
}

func (s *PreparerSuite) Test_SelectBlueprintFormat_succeeds_when_other_file_missing() {
	// Create only yaml file
	yamlPath := filepath.Join(s.tempDir, "project.blueprint.yaml.tmpl")
	err := os.WriteFile(yamlPath, []byte("version: 1.0"), 0644)
	s.Require().NoError(err)

	// Select yaml format (jsonc doesn't exist but that's fine)
	err = s.preparer.SelectBlueprintFormat(s.tempDir, "yaml")
	s.NoError(err)
}

// SubstitutePlaceholders tests

func (s *PreparerSuite) Test_SubstitutePlaceholders_processes_tmpl_files() {
	// Create a template file
	tmplPath := filepath.Join(s.tempDir, "README.md.tmpl")
	tmplContent := `# {{.ProjectName}}

Welcome to {{.NormalisedProjectName}}!

Format: {{.BlueprintFormat}}
`
	err := os.WriteFile(tmplPath, []byte(tmplContent), 0644)
	s.Require().NoError(err)

	// Process templates
	values := NewTemplateValues("My Project", "yaml")
	err = s.preparer.SubstitutePlaceholders(s.tempDir, values)
	s.NoError(err)

	// Verify .tmpl file is removed
	_, err = os.Stat(tmplPath)
	s.True(os.IsNotExist(err))

	// Verify output file exists with correct content
	outputPath := filepath.Join(s.tempDir, "README.md")
	content, err := os.ReadFile(outputPath)
	s.NoError(err)

	s.Contains(string(content), "# My Project")
	s.Contains(string(content), "Welcome to my-project!")
	s.Contains(string(content), "Format: yaml")
}

func (s *PreparerSuite) Test_SubstitutePlaceholders_processes_nested_tmpl_files() {
	// Create nested directory structure
	subDir := filepath.Join(s.tempDir, "config")
	err := os.Mkdir(subDir, 0755)
	s.Require().NoError(err)

	// Create template files in nested directory
	tmplPath := filepath.Join(subDir, "settings.json.tmpl")
	tmplContent := `{"name": "{{.ProjectName}}"}`
	err = os.WriteFile(tmplPath, []byte(tmplContent), 0644)
	s.Require().NoError(err)

	// Process templates
	values := NewTemplateValues("TestApp", "jsonc")
	err = s.preparer.SubstitutePlaceholders(s.tempDir, values)
	s.NoError(err)

	// Verify output file exists
	outputPath := filepath.Join(subDir, "settings.json")
	content, err := os.ReadFile(outputPath)
	s.NoError(err)
	s.Equal(`{"name": "TestApp"}`, string(content))
}

func (s *PreparerSuite) Test_SubstitutePlaceholders_skips_git_directory() {
	// Create .git directory with a .tmpl file
	gitDir := filepath.Join(s.tempDir, ".git")
	err := os.Mkdir(gitDir, 0755)
	s.Require().NoError(err)

	tmplPath := filepath.Join(gitDir, "config.tmpl")
	err = os.WriteFile(tmplPath, []byte("{{.ProjectName}}"), 0644)
	s.Require().NoError(err)

	// Process templates
	values := NewTemplateValues("Test", "yaml")
	err = s.preparer.SubstitutePlaceholders(s.tempDir, values)
	s.NoError(err)

	// Verify .git template file still exists (was skipped)
	_, err = os.Stat(tmplPath)
	s.NoError(err)
}

func (s *PreparerSuite) Test_SubstitutePlaceholders_succeeds_with_no_tmpl_files() {
	// Create a non-template file
	filePath := filepath.Join(s.tempDir, "README.md")
	err := os.WriteFile(filePath, []byte("# Test"), 0644)
	s.Require().NoError(err)

	// Process templates (none exist)
	values := NewTemplateValues("Test", "yaml")
	err = s.preparer.SubstitutePlaceholders(s.tempDir, values)
	s.NoError(err)

	// Verify non-template file is unchanged
	content, err := os.ReadFile(filePath)
	s.NoError(err)
	s.Equal("# Test", string(content))
}

func (s *PreparerSuite) Test_SubstitutePlaceholders_preserves_file_permissions() {
	// Create a template file with executable permission
	tmplPath := filepath.Join(s.tempDir, "script.sh.tmpl")
	err := os.WriteFile(tmplPath, []byte("#!/bin/bash\necho {{.ProjectName}}"), 0755)
	s.Require().NoError(err)

	// Process templates
	values := NewTemplateValues("Test", "yaml")
	err = s.preparer.SubstitutePlaceholders(s.tempDir, values)
	s.NoError(err)

	// Verify output file has same permissions
	outputPath := filepath.Join(s.tempDir, "script.sh")
	info, err := os.Stat(outputPath)
	s.NoError(err)
	s.Equal(os.FileMode(0755), info.Mode().Perm())
}

func (s *PreparerSuite) Test_SubstitutePlaceholders_returns_error_for_invalid_template() {
	// Create a template file with invalid syntax
	tmplPath := filepath.Join(s.tempDir, "bad.tmpl")
	err := os.WriteFile(tmplPath, []byte("{{.Invalid"), 0644)
	s.Require().NoError(err)

	// Process templates
	values := NewTemplateValues("Test", "yaml")
	err = s.preparer.SubstitutePlaceholders(s.tempDir, values)
	s.Error(err)
	s.Contains(err.Error(), "failed to parse template")
}

// NewTemplateValues tests

func (s *PreparerSuite) Test_NewTemplateValues_creates_correct_values() {
	values := NewTemplateValues("My Project", "yaml")

	s.Equal("My Project", values.ProjectName)
	s.Equal("my-project", values.NormalisedProjectName)
	s.Equal("yaml", values.BlueprintFormat)
}

// normaliseProjectName tests

func (s *PreparerSuite) Test_normaliseProjectName_handles_spaces() {
	s.Equal("my-project", normaliseProjectName("My Project"))
	s.Equal("hello-world", normaliseProjectName("Hello World"))
}

func (s *PreparerSuite) Test_normaliseProjectName_handles_camelCase() {
	s.Equal("my-project", normaliseProjectName("MyProject"))
	s.Equal("hello-world-app", normaliseProjectName("HelloWorldApp"))
}

func (s *PreparerSuite) Test_normaliseProjectName_handles_underscores() {
	s.Equal("my-project", normaliseProjectName("my_project"))
	s.Equal("hello-world", normaliseProjectName("hello_world"))
}

func (s *PreparerSuite) Test_normaliseProjectName_handles_mixed_cases() {
	s.Equal("my-cool-project", normaliseProjectName("My Cool_Project"))
	s.Equal("apiserver-v2", normaliseProjectName("APIServer_v2"))
}

func (s *PreparerSuite) Test_normaliseProjectName_removes_consecutive_hyphens() {
	s.Equal("my-project", normaliseProjectName("my--project"))
	s.Equal("hello-world", normaliseProjectName("hello---world"))
}

func (s *PreparerSuite) Test_normaliseProjectName_trims_hyphens() {
	s.Equal("my-project", normaliseProjectName("-my-project-"))
	s.Equal("hello", normaliseProjectName("---hello---"))
}

func (s *PreparerSuite) Test_normaliseProjectName_lowercase() {
	s.Equal("myproject", normaliseProjectName("MYPROJECT"))
	s.Equal("test", normaliseProjectName("TEST"))
}

func TestPreparerSuite(t *testing.T) {
	suite.Run(t, new(PreparerSuite))
}
