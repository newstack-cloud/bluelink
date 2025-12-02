package sharedui

import (
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/consts"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

type SelectBlueprintModel struct {
	start           tea.Model
	source          tea.Model
	selectLocalFile tea.Model
	inputRemoteFile tea.Model
	selectedSource  string
	selectedFile    string
	autoSelect      bool
	stage           selectBlueprintStage
	quitting        bool
	err             error
}

type selectBlueprintStage int

const (
	// Stage where the user starts the select blueprint process.
	// The user will be able to select from options to use the default blueprint file
	// or to choose a file from a local or remote file.
	selectBlueprintStageStart selectBlueprintStage = iota

	// Stage where the user selects the source of the blueprint file.
	// Can be one of the following:
	// - "file" (local file)
	// - "https" (public URL)
	// - "s3" (AWS S3)
	// - "gcs" (Google Cloud Storage)
	// - "azureblob" (Azure Blob Storage)
	selectBlueprintStageSelectSource

	// Stage where the user inputs the location of the file
	// relative to a remote source scheme.
	selectBlueprintStageInputFileLocation

	// Stage where the user selects a local file.
	selectBlueprintStageSelectLocalFile

	// Stage where the user has selected a blueprint file.
	selectBlueprintStageSelected
)

const listHeight = 14

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m SelectBlueprintModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	fcmd := m.selectLocalFile.Init()
	rcmd := m.inputRemoteFile.Init()
	cmds = append(cmds, fcmd, rcmd)

	if m.autoSelect {
		// Dispatch command to select the blueprint file
		// so the validation model can trigger the validation process.
		cmds = append(cmds, selectBlueprintCmd(m.selectedFile, m.selectedSource))
	}

	return tea.Batch(cmds...)
}

func (m SelectBlueprintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.stage == selectBlueprintStageSelectLocalFile {
			var selectLocalFileCmd tea.Cmd
			m.selectLocalFile, selectLocalFileCmd = m.selectLocalFile.Update(msg)
			cmds = append(cmds, selectLocalFileCmd)
		}
	case SelectBlueprintStartMsg:
		m.stage = stageFromStartSelection(msg.Selection)
		if m.stage == selectBlueprintStageSelected {
			cmds = append(cmds, selectBlueprintCmd(m.selectedFile, m.selectedSource))
		}
	case SelectBlueprintSourceMsg:
		m.stage = stageFromSource(msg.Source)
		m.selectedSource = msg.Source
		// Ensure the remote file input form is updated to guide the user to input the location
		// based on the source.
		var inputRemoteFileCmd tea.Cmd
		m.inputRemoteFile, inputRemoteFileCmd = m.inputRemoteFile.Update(msg)
		cmds = append(cmds, inputRemoteFileCmd)
	case SelectBlueprintMsg:
		m.selectedFile = msg.BlueprintFile
		m.stage = selectBlueprintStageSelected
	case clearErrorMsg:
		m.err = nil
	case tea.WindowSizeMsg:
		// Ensure all models are updated
		// when the window size changes so they fit the new window size when displayed.
		var startCmd tea.Cmd
		m.start, startCmd = m.start.Update(msg)
		cmds = append(cmds, startCmd)

		var sourceCmd tea.Cmd
		m.source, sourceCmd = m.source.Update(msg)
		cmds = append(cmds, sourceCmd)

		var selectLocalFileCmd tea.Cmd
		m.selectLocalFile, selectLocalFileCmd = m.selectLocalFile.Update(msg)
		cmds = append(cmds, selectLocalFileCmd)
	default:
		// Ensure the local file selection model is updated for other messages to make sure it picks
		// up the message dispatched from initialising the file picker.
		var selectLocalFileCmd tea.Cmd
		m.selectLocalFile, selectLocalFileCmd = m.selectLocalFile.Update(msg)
		cmds = append(cmds, selectLocalFileCmd)
	}

	switch m.stage {
	case selectBlueprintStageStart:
		var startCmd tea.Cmd
		m.start, startCmd = m.start.Update(msg)
		cmds = append(cmds, startCmd)
	case selectBlueprintStageSelectSource:
		var sourceCmd tea.Cmd
		m.source, sourceCmd = m.source.Update(msg)
		cmds = append(cmds, sourceCmd)
	case selectBlueprintStageInputFileLocation:
		var inputRemoteFileCmd tea.Cmd
		m.inputRemoteFile, inputRemoteFileCmd = m.inputRemoteFile.Update(msg)
		cmds = append(cmds, inputRemoteFileCmd)
	}

	return m, tea.Batch(cmds...)
}

func (m SelectBlueprintModel) View() string {
	if m.quitting {
		return ""
	}

	var s strings.Builder

	switch m.stage {
	case selectBlueprintStageStart:
		s.WriteString("\n\n")
		s.WriteString(fmt.Sprintf("    The default blueprint file is: %s \n\n", m.selectedFile))
		s.WriteString(m.start.View() + "\n")
	case selectBlueprintStageSelectSource:
		s.WriteString("\n\n" + m.source.View() + "\n")
	case selectBlueprintStageSelectLocalFile:
		s.WriteString("\n\n" + m.selectLocalFile.View() + "\n")
	case selectBlueprintStageInputFileLocation:
		s.WriteString("\n\n" + m.inputRemoteFile.View() + "\n")
	case selectBlueprintStageSelected:
		fullBlueprintPath := ToFullBlueprintPath(m.selectedFile, m.selectedSource)
		s.WriteString("\n\nBlueprint file selected: " + fullBlueprintPath + "\n")
	}

	return s.String()
}

func NewSelectBlueprint(
	blueprintFile string,
	autoSelect bool,
	action string,
	bluelinkStyles *styles.BluelinkStyles,
	fp *filepicker.Model,
) (*SelectBlueprintModel, error) {
	selectLocalFile, err := NewSelectBlueprintLocalFile(fp, bluelinkStyles)
	if err != nil {
		return nil, err
	}

	start := NewSelect(
		"Get started by choosing a blueprint file",
		selectBlueprintStartListItems(),
		bluelinkStyles,
		selectBlueprintStartCmd,
		false, // enableFiltering
	)
	source := NewSelect(
		fmt.Sprintf("Where is the blueprint that you want to %s stored?", action),
		blueprintSourceListItems(),
		bluelinkStyles,
		selectBlueprintSourceCmd,
		false, // enableFiltering
	)
	inputRemoteFile := NewSelectBlueprintRemoteFile(blueprintFile, bluelinkStyles)

	defaultFile, defaultSource := initialFileAndSourceFromBlueprintFile(blueprintFile)
	// Only join with current directory for local file sources.
	// Remote sources (S3, GCS, Azure Blob, HTTPS) should keep their path as-is.
	selectedFile := defaultFile
	if defaultSource == consts.BlueprintSourceFile {
		selectedFile = filepath.Join(fp.CurrentDirectory, defaultFile)
	}

	return &SelectBlueprintModel{
		selectLocalFile: selectLocalFile,
		inputRemoteFile: inputRemoteFile,
		autoSelect:      autoSelect,
		start:           start,
		source:          source,
		selectedSource:  defaultSource,
		selectedFile:    selectedFile,
		stage:           selectBlueprintStageStart,
	}, nil
}

func initialFileAndSourceFromBlueprintFile(blueprintFile string) (string, string) {
	if strings.HasPrefix(blueprintFile, "https://") {
		return blueprintFile, consts.BlueprintSourceHTTPS
	}

	if withoutScheme, ok := strings.CutPrefix(blueprintFile, "s3://"); ok {
		return withoutScheme, consts.BlueprintSourceS3
	}

	if withoutScheme, ok := strings.CutPrefix(blueprintFile, "gcs://"); ok {
		return withoutScheme, consts.BlueprintSourceGCS
	}

	if withoutScheme, ok := strings.CutPrefix(blueprintFile, "azureblob://"); ok {
		return withoutScheme, consts.BlueprintSourceAzureBlob
	}

	return blueprintFile, consts.BlueprintSourceFile
}

func stageFromSource(source string) selectBlueprintStage {
	switch source {
	case consts.BlueprintSourceFile:
		return selectBlueprintStageSelectLocalFile
	case consts.BlueprintSourceS3:
		return selectBlueprintStageInputFileLocation
	case consts.BlueprintSourceGCS:
		return selectBlueprintStageInputFileLocation
	case consts.BlueprintSourceAzureBlob:
		return selectBlueprintStageInputFileLocation
	case consts.BlueprintSourceHTTPS:
		return selectBlueprintStageInputFileLocation
	}

	return selectBlueprintStageSelectSource
}

func stageFromStartSelection(selection string) selectBlueprintStage {
	switch selection {
	case "default":
		return selectBlueprintStageSelected
	case "select":
		return selectBlueprintStageSelectSource
	}

	return selectBlueprintStageStart
}

// ToFullBlueprintPath returns the full path to the blueprint file based on the source.
func ToFullBlueprintPath(blueprintFile, source string) string {
	switch source {
	case consts.BlueprintSourceS3:
		return "s3://" + blueprintFile
	case consts.BlueprintSourceGCS:
		return "gcs://" + blueprintFile
	case consts.BlueprintSourceAzureBlob:
		return "azureblob://" + blueprintFile
	}

	return blueprintFile
}
