package sharedui

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/exp/teatest"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/consts"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/testutils"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
	"github.com/stretchr/testify/suite"
)

type SelectBlueprintSuite struct {
	suite.Suite
	tempDir string
}

func (s *SelectBlueprintSuite) SetupSuite() {
	s.tempDir = s.T().TempDir()

	files, err := os.ReadDir("__testdata")
	s.NoError(err)

	for _, file := range files {
		fileData, err := os.ReadFile(filepath.Join("__testdata", file.Name()))
		s.NoError(err)
		os.WriteFile(filepath.Join(s.tempDir, file.Name()), fileData, 0644)
	}
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_local_file() {
	styles := styles.NewBluelinkStyles(lipgloss.NewRenderer(os.Stdout))
	fp, err := BlueprintLocalFilePicker(styles)
	s.NoError(err)

	// Make it so the file picker is in a temporary directory
	// with a text file and a blueprint file.
	fp.CurrentDirectory = s.tempDir
	fp.DirAllowed = false
	selectModel, err := NewSelectBlueprint(
		"",
		/* autoValidate */ false,
		"validate",
		styles,
		&fp,
	)
	s.NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		selectModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select the second option that is for selecting a file instead of using the default
	// blueprint file.
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Where is the blueprint that you want to validate stored?",
	)

	// Select the first option that is for picking a local file.
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Pick a blueprint file:",
	)

	// Try to select the first file which is not a valid blueprint file.
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"other.txt is not a valid blueprint file.",
	)

	// Select the second option which is the valid blueprint file.
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	blueprintPath := filepath.Join(s.tempDir, "project.blueprint.yml")
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		fmt.Sprintf("Blueprint file selected: %s", blueprintPath),
	)

	err = testModel.Quit()
	s.NoError(err)

	finalModel, isSelectBlueprintModel := testModel.FinalModel(s.T()).(SelectBlueprintModel)
	s.Assert().True(isSelectBlueprintModel)
	s.Assert().Equal(blueprintPath, finalModel.selectedFile)
	s.Assert().Equal(consts.BlueprintSourceFile, finalModel.selectedSource)
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_s3_file() {
	styles := styles.NewBluelinkStyles(lipgloss.NewRenderer(os.Stdout))
	fp, err := BlueprintLocalFilePicker(styles)
	s.NoError(err)

	selectModel, err := NewSelectBlueprint(
		"",
		/* autoValidate */ false,
		"validate",
		styles,
		&fp,
	)
	s.NoError(err)

	testModel := teatest.NewTestModel(
		s.T(),
		selectModel,
		teatest.WithInitialTermSize(300, 100),
	)

	// Select the second option that is for selecting a file instead of using the default
	// blueprint file.
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"Where is the blueprint that you want to validate stored?",
	)

	// Select the second option that is for picking a remote file from S3.
	testutils.KeyDown(testModel)
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		"S3 bucket",
	)

	testModel.Type("test-bucket")
	testutils.KeyEnter(testModel)

	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		// "enter submit" is the help text displayed when on the final step of the form.
		"enter submit",
	)

	testModel.Type("project1/project.blueprint.yml")
	testutils.KeyEnter(testModel)

	blueprintPath := "test-bucket/project1/project.blueprint.yml"
	testutils.WaitForContains(
		s.T(),
		testModel.Output(),
		fmt.Sprintf("Blueprint file selected: s3://%s", blueprintPath),
	)

	err = testModel.Quit()
	s.NoError(err)

	finalModel, isSelectBlueprintModel := testModel.FinalModel(s.T()).(SelectBlueprintModel)
	s.Assert().True(isSelectBlueprintModel)
	s.Assert().Equal(blueprintPath, finalModel.selectedFile)
	s.Assert().Equal(consts.BlueprintSourceS3, finalModel.selectedSource)
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_gcs_file() {
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_https_file() {
}

func (s *SelectBlueprintSuite) Test_select_blueprint_from_remote_azure_blob_file() {
}

func TestSelectBlueprintSuite(t *testing.T) {
	suite.Run(t, new(SelectBlueprintSuite))
}
