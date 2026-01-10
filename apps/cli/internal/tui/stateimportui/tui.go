package stateimportui

import (
	"io"
	"os"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/spinner"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/stateio"
	"github.com/newstack-cloud/deploy-cli-sdk/consts"
	stylespkg "github.com/newstack-cloud/deploy-cli-sdk/styles"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

var (
	quitTextStyle = stylespkg.NewDefaultStyles(
		stylespkg.NewBluelinkPalette(),
	).Muted.Margin(1, 0, 2, 4)
)

func createFilePicker(styles *stylespkg.Styles, allowedTypes []string) (filepicker.Model, error) {
	fp := filepicker.New()
	fpStyles := filepicker.DefaultStyles()
	fpStyles.Selected = styles.Selected
	fpStyles.File = styles.Selectable
	fpStyles.Directory = styles.Selectable
	fpStyles.Cursor = styles.Selected
	fp.Styles = fpStyles
	fp.AllowedTypes = allowedTypes

	currentDir, err := os.Getwd()
	if err != nil {
		return filepicker.Model{}, err
	}
	fp.CurrentDirectory = currentDir

	return fp, nil
}

func createSelectFileModel(
	styles *stylespkg.Styles,
	filePath string,
	autoSelect bool,
) (*sharedui.SelectFileModel, error) {
	allowedTypes := []string{".json"}
	fileTypeName := "state file"

	fp, err := createFilePicker(styles, allowedTypes)
	if err != nil {
		return nil, err
	}

	return sharedui.NewSelectFile(sharedui.SelectFileConfig{
		InitialFile:  filePath,
		AutoSelect:   autoSelect,
		Action:       "import",
		Styles:       styles,
		FilePicker:   &fp,
		FileTypeName: fileTypeName,
		SupportedSources: []string{
			consts.FileSourceLocal,
			consts.FileSourceS3,
			consts.FileSourceGCS,
			consts.FileSourceAzureBlob,
		},
	})
}

type stateImportSessionState uint32

const (
	stateImportFileSelect stateImportSessionState = iota
	stateImportRunning
	stateImportComplete
)

// MainModel is the top-level model for the state import command TUI.
type MainModel struct {
	sessionState stateImportSessionState
	filePath     string
	quitting     bool
	selectFile   tea.Model
	importModel  *ImportModel
	styles       *stylespkg.Styles
	width        int
	Error        error
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	importCmd := m.importModel.Init()
	cmds = append(cmds, importCmd)

	if m.selectFile != nil {
		fileCmd := m.selectFile.Init()
		cmds = append(cmds, fileCmd)
	}

	// If we're starting in running state (auto-import mode), start the import
	if m.sessionState == stateImportRunning {
		cmds = append(cmds, m.importModel.StartImport())
	}

	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case sharedui.SelectFileMsg:
		m.filePath = sharedui.ToFullFilePath(msg.File, msg.Source)
		m.sessionState = stateImportRunning
		m.importModel.SetFilePath(m.filePath)
		cmds = append(cmds, m.importModel.StartImport())

	case sharedui.ClearSelectedFileMsg:
		m.sessionState = stateImportFileSelect
		m.filePath = ""

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.importModel.SetWidth(msg.Width)
		if m.selectFile != nil {
			var fileCmd tea.Cmd
			m.selectFile, fileCmd = m.selectFile.Update(msg)
			cmds = append(cmds, fileCmd)
		}

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			m.quitting = true
			return m, tea.Quit
		case "q":
			// Quit if import is complete and finished, or if there's an error
			if (m.sessionState == stateImportComplete && m.importModel.IsFinished()) || m.Error != nil {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.importModel, cmd = m.importModel.Update(msg)
		cmds = append(cmds, cmd)

	case ImportCompleteMsg:
		m.sessionState = stateImportComplete
		var cmd tea.Cmd
		m.importModel, cmd = m.importModel.Update(msg)
		cmds = append(cmds, cmd)
		if msg.Err != nil {
			m.Error = msg.Err
		}
		// Auto-quit in headless mode after import completes
		if m.importModel.headless {
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch m.sessionState {
	case stateImportFileSelect:
		if m.selectFile != nil {
			newSelectFile, newCmd := m.selectFile.Update(msg)
			selectFileModel, ok := newSelectFile.(sharedui.SelectFileModel)
			if !ok {
				panic("failed to perform assertion on select file model in state import")
			}
			m.selectFile = selectFileModel
			cmds = append(cmds, newCmd)
		}
	case stateImportRunning, stateImportComplete:
		var cmd tea.Cmd
		m.importModel, cmd = m.importModel.Update(msg)
		cmds = append(cmds, cmd)
		if m.importModel.err != nil {
			m.Error = m.importModel.err
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}

	switch m.sessionState {
	case stateImportFileSelect:
		if m.selectFile != nil {
			return m.selectFile.View()
		}
		return ""
	case stateImportRunning, stateImportComplete:
		selected := "\n  Importing from: " + m.styles.Selected.Render(m.filePath) + "\n"
		return selected + m.importModel.View()
	}

	return ""
}

// StateImportAppConfig holds configuration for creating a new state import app.
type StateImportAppConfig struct {
	FilePath       string
	EngineConfig   *stateio.EngineConfig
	Styles         *stylespkg.Styles
	Headless       bool
	HeadlessWriter io.Writer
	JSONMode       bool
}

// NewStateImportApp creates a new state import application.
func NewStateImportApp(config StateImportAppConfig) (*MainModel, error) {
	// Determine if we're in auto-import mode (headless or file provided)
	autoImport := config.FilePath != "" || config.Headless

	// Determine the initial session state and create appropriate sub-models
	var sessionState stateImportSessionState
	var selectFile tea.Model

	if autoImport {
		// In headless/auto mode, go straight to running
		sessionState = stateImportRunning
	} else {
		// Interactive mode - start with file selection
		sessionState = stateImportFileSelect
		var err error
		selectFile, err = createSelectFileModel(config.Styles, "", false)
		if err != nil {
			return nil, err
		}
	}

	importModel := NewImportModel(ImportModelConfig{
		EngineConfig:   config.EngineConfig,
		FilePath:       config.FilePath,
		Styles:         config.Styles,
		Headless:       config.Headless,
		HeadlessWriter: config.HeadlessWriter,
		JSONMode:       config.JSONMode,
	})

	return &MainModel{
		sessionState: sessionState,
		filePath:     config.FilePath,
		selectFile:   selectFile,
		importModel:  importModel,
		styles:       config.Styles,
		width:        80, // Default width, will be updated on first WindowSizeMsg
	}, nil
}
