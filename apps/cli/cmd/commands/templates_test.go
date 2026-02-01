package commands

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/suite"
)

type TemplatesCommandSuite struct {
	suite.Suite
	tempDir     string
	originalDir string
}

func (s *TemplatesCommandSuite) SetupTest() {
	tempDir, err := os.MkdirTemp("", "templates-cmd-test-*")
	s.Require().NoError(err)
	s.tempDir = tempDir

	err = os.WriteFile(
		filepath.Join(tempDir, "bluelink.config.toml"),
		[]byte(""),
		0644,
	)
	s.Require().NoError(err)

	s.originalDir, err = os.Getwd()
	s.Require().NoError(err)
	os.Chdir(tempDir)
}

func (s *TemplatesCommandSuite) TearDownTest() {
	os.Chdir(s.originalDir)
	os.RemoveAll(s.tempDir)
}

func (s *TemplatesCommandSuite) Test_templates_command_exists() {
	rootCmd := NewRootCmd()
	templatesCmd, _, err := rootCmd.Find([]string{"templates"})

	s.NoError(err)
	s.NotNil(templatesCmd)
	s.Equal("templates", templatesCmd.Use)
}

func (s *TemplatesCommandSuite) Test_templates_list_command_exists() {
	rootCmd := NewRootCmd()
	listCmd, _, err := rootCmd.Find([]string{"templates", "list"})

	s.NoError(err)
	s.NotNil(listCmd)
	s.Equal("list", listCmd.Use)
}

func (s *TemplatesCommandSuite) Test_templates_help_contains_description() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"templates", "--help"})

	rootCmd.Execute()
	output := buf.String()

	s.Contains(output, "templates")
	s.Contains(output, "list")
}

func (s *TemplatesCommandSuite) Test_templates_list_help_contains_usage() {
	rootCmd := NewRootCmd()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"templates", "list", "--help"})

	rootCmd.Execute()
	output := buf.String()

	s.Contains(output, "list")
	s.Contains(output, "--search")
	s.Contains(output, "Usage:")
}

func (s *TemplatesCommandSuite) Test_templates_list_has_search_flag() {
	rootCmd := NewRootCmd()
	listCmd, _, err := rootCmd.Find([]string{"templates", "list"})

	s.NoError(err)
	s.NotNil(listCmd)

	searchFlag := listCmd.Flag("search")
	s.NotNil(searchFlag)
	s.Equal("", searchFlag.DefValue)
}

func (s *TemplatesCommandSuite) Test_templates_list_accepts_no_args() {
	rootCmd := NewRootCmd()
	listCmd, _, err := rootCmd.Find([]string{"templates", "list"})

	s.NoError(err)
	s.NotNil(listCmd)

	err = listCmd.Args(listCmd, []string{})
	s.NoError(err)
}

func (s *TemplatesCommandSuite) Test_templates_list_rejects_args() {
	rootCmd := NewRootCmd()
	listCmd, _, err := rootCmd.Find([]string{"templates", "list"})

	s.NoError(err)
	s.NotNil(listCmd)

	err = listCmd.Args(listCmd, []string{"unexpected-arg"})
	s.Error(err)
}

func TestTemplatesCommandSuite(t *testing.T) {
	suite.Run(t, new(TemplatesCommandSuite))
}
