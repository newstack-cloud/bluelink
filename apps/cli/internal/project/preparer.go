package project

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"text/template"
)

// Preparer is an interface that provides methods for preparing a project
// after it has been cloned from a template repository.
type Preparer interface {
	// RemoveGitHistory removes the .git directory from the project.
	RemoveGitHistory(directory string) error
	// RemoveMaintainerFiles removes files that are meant for template maintainers
	// (e.g., README.md, LICENSE) before template processing.
	RemoveMaintainerFiles(directory string) error
	// SelectBlueprintFormat removes the unused blueprint file.
	// The selected format's .tmpl file will be processed by SubstitutePlaceholders.
	SelectBlueprintFormat(directory string, format string) error
	// SubstitutePlaceholders finds all .tmpl files, processes them as Go templates,
	// writes the output without the .tmpl extension, and removes the original .tmpl files.
	SubstitutePlaceholders(directory string, values TemplateValues) error
}

// TemplateValues holds the values to substitute in template files.
type TemplateValues struct {
	ProjectName           string
	NormalisedProjectName string
	BlueprintFormat       string
}

// NewTemplateValues creates a TemplateValues struct with computed fields.
func NewTemplateValues(projectName, blueprintFormat string) TemplateValues {
	return TemplateValues{
		ProjectName:           projectName,
		NormalisedProjectName: normaliseProjectName(projectName),
		BlueprintFormat:       blueprintFormat,
	}
}

// normaliseProjectName converts a project name to a lowercase, hyphenated format.
// For example: "My Project" -> "my-project", "MyProject" -> "my-project"
func normaliseProjectName(name string) string {
	// Insert hyphens before uppercase letters (for camelCase/PascalCase)
	re := regexp.MustCompile(`([a-z])([A-Z])`)
	name = re.ReplaceAllString(name, "${1}-${2}")

	// Replace spaces and underscores with hyphens
	name = strings.ReplaceAll(name, " ", "-")
	name = strings.ReplaceAll(name, "_", "-")

	// Convert to lowercase
	name = strings.ToLower(name)

	// Remove consecutive hyphens
	re = regexp.MustCompile(`-+`)
	name = re.ReplaceAllString(name, "-")

	// Trim leading/trailing hyphens
	name = strings.Trim(name, "-")

	return name
}

type preparerImpl struct{}

// NewDefaultPreparer creates a new instance of the default Preparer implementation.
func NewDefaultPreparer() Preparer {
	return &preparerImpl{}
}

// RemoveGitHistory removes the .git directory from the project.
func (p *preparerImpl) RemoveGitHistory(directory string) error {
	gitDir := filepath.Join(directory, ".git")
	if err := os.RemoveAll(gitDir); err != nil {
		return fmt.Errorf("failed to remove .git directory: %w", err)
	}
	return nil
}

// maintainerFiles lists files that are meant for template maintainers
// and should be removed before template processing.
var maintainerFiles = []string{
	"README.md",
	"LICENSE",
}

// RemoveMaintainerFiles removes files that are meant for template maintainers.
// These files exist in the template repository for documentation purposes
// but should not be included in generated projects (they will be replaced
// by processed .tmpl files like README.md.tmpl -> README.md).
func (p *preparerImpl) RemoveMaintainerFiles(directory string) error {
	for _, filename := range maintainerFiles {
		filePath := filepath.Join(directory, filename)

		// Skip if file doesn't exist
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			continue
		}

		if err := os.Remove(filePath); err != nil {
			return fmt.Errorf("failed to remove maintainer file %s: %w", filename, err)
		}
	}
	return nil
}

// SelectBlueprintFormat removes the unused blueprint template file.
// Template files use the naming convention: project.blueprint.{format}.tmpl
func (p *preparerImpl) SelectBlueprintFormat(directory string, format string) error {
	// Define the blueprint template file names
	yamlFile := filepath.Join(directory, "project.blueprint.yaml.tmpl")
	jsoncFile := filepath.Join(directory, "project.blueprint.jsonc.tmpl")

	// Determine which file to remove based on selected format
	var fileToRemove string
	var fileToKeep string
	switch format {
	case "yaml":
		fileToRemove = jsoncFile
		fileToKeep = yamlFile
	case "jsonc":
		fileToRemove = yamlFile
		fileToKeep = jsoncFile
	default:
		return fmt.Errorf("unsupported blueprint format: %s", format)
	}

	// Check if the file to keep exists
	if _, err := os.Stat(fileToKeep); os.IsNotExist(err) {
		return fmt.Errorf("blueprint template for format %q not found: %s", format, fileToKeep)
	}

	// Remove the unused blueprint file (ignore if it doesn't exist)
	if _, err := os.Stat(fileToRemove); err == nil {
		if err := os.Remove(fileToRemove); err != nil {
			return fmt.Errorf("failed to remove unused blueprint file: %w", err)
		}
	}

	return nil
}

// SubstitutePlaceholders finds all .tmpl files in the directory, processes them
// as Go templates with the provided values, writes the output to a new file
// without the .tmpl extension, and removes the original .tmpl file.
func (p *preparerImpl) SubstitutePlaceholders(directory string, values TemplateValues) error {
	// Find all .tmpl files in the directory
	tmplFiles, err := p.findTemplateFiles(directory)
	if err != nil {
		return fmt.Errorf("failed to find template files: %w", err)
	}

	for _, tmplPath := range tmplFiles {
		if err := p.processTemplateFile(tmplPath, values); err != nil {
			relPath, _ := filepath.Rel(directory, tmplPath)
			return fmt.Errorf("failed to process %s: %w", relPath, err)
		}
	}

	return nil
}

// findTemplateFiles recursively finds all .tmpl files in the given directory.
func (p *preparerImpl) findTemplateFiles(directory string) ([]string, error) {
	var tmplFiles []string

	err := filepath.Walk(directory, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip .git directory
		if info.IsDir() && info.Name() == ".git" {
			return filepath.SkipDir
		}

		// Collect .tmpl files
		if !info.IsDir() && strings.HasSuffix(info.Name(), ".tmpl") {
			tmplFiles = append(tmplFiles, path)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return tmplFiles, nil
}

// processTemplateFile reads a .tmpl file, processes it as a Go template,
// writes the output to a new file without the .tmpl extension, and removes
// the original .tmpl file.
func (p *preparerImpl) processTemplateFile(tmplPath string, values TemplateValues) error {
	content, err := os.ReadFile(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	tmpl, err := template.New(filepath.Base(tmplPath)).Parse(string(content))
	if err != nil {
		return fmt.Errorf("failed to parse template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, values); err != nil {
		return fmt.Errorf("failed to execute template: %w", err)
	}

	outputPath := strings.TrimSuffix(tmplPath, ".tmpl")

	info, err := os.Stat(tmplPath)
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}

	if err := os.WriteFile(outputPath, buf.Bytes(), info.Mode()); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	if err := os.Remove(tmplPath); err != nil {
		return fmt.Errorf("failed to remove template file: %w", err)
	}

	return nil
}
