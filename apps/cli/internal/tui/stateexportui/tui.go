package stateexportui

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
	fileTypeName := "output file"

	fp, err := createFilePicker(styles, allowedTypes)
	if err != nil {
		return nil, err
	}

	return sharedui.NewSelectFile(sharedui.SelectFileConfig{
		InitialFile:  filePath,
		AutoSelect:   autoSelect,
		Action:       "export",
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

type stateExportSessionState uint32

const (
	stateExportFileSelect stateExportSessionState = iota
	stateExportRunning
	stateExportComplete
)

// MainModel is the top-level model for the state export command TUI.
type MainModel struct {
	sessionState stateExportSessionState
	filePath     string
	quitting     bool
	selectFile   tea.Model
	exportModel  *ExportModel
	styles       *stylespkg.Styles
	width        int
	Error        error
}

func (m MainModel) Init() tea.Cmd {
	cmds := []tea.Cmd{}

	exportCmd := m.exportModel.Init()
	cmds = append(cmds, exportCmd)

	if m.selectFile != nil {
		fileCmd := m.selectFile.Init()
		cmds = append(cmds, fileCmd)
	}

	// If we're starting in running state (auto-export mode), start the export
	if m.sessionState == stateExportRunning {
		cmds = append(cmds, m.exportModel.StartExport())
	}

	return tea.Batch(cmds...)
}

func (m MainModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}

	switch msg := msg.(type) {
	case sharedui.SelectFileMsg:
		m.filePath = sharedui.ToFullFilePath(msg.File, msg.Source)
		m.sessionState = stateExportRunning
		m.exportModel.SetFilePath(m.filePath)
		cmds = append(cmds, m.exportModel.StartExport())

	case sharedui.ClearSelectedFileMsg:
		m.sessionState = stateExportFileSelect
		m.filePath = ""

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.exportModel.SetWidth(msg.Width)
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
			if (m.sessionState == stateExportComplete && m.exportModel.IsFinished()) || m.Error != nil {
				m.quitting = true
				return m, tea.Quit
			}
		}

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.exportModel, cmd = m.exportModel.Update(msg)
		cmds = append(cmds, cmd)

	case ExportCompleteMsg:
		m.sessionState = stateExportComplete
		var cmd tea.Cmd
		m.exportModel, cmd = m.exportModel.Update(msg)
		cmds = append(cmds, cmd)
		if msg.Err != nil {
			m.Error = msg.Err
		}
		if m.exportModel.headless {
			m.quitting = true
			return m, tea.Quit
		}
	}

	switch m.sessionState {
	case stateExportFileSelect:
		if m.selectFile != nil {
			newSelectFile, newCmd := m.selectFile.Update(msg)
			selectFileModel, ok := newSelectFile.(sharedui.SelectFileModel)
			if !ok {
				panic("failed to perform assertion on select file model in state export")
			}
			m.selectFile = selectFileModel
			cmds = append(cmds, newCmd)
		}
	case stateExportRunning, stateExportComplete:
		var cmd tea.Cmd
		m.exportModel, cmd = m.exportModel.Update(msg)
		cmds = append(cmds, cmd)
		if m.exportModel.err != nil {
			m.Error = m.exportModel.err
		}
	}

	return m, tea.Batch(cmds...)
}

func (m MainModel) View() string {
	if m.quitting {
		return quitTextStyle.Render("See you next time.")
	}

	switch m.sessionState {
	case stateExportFileSelect:
		if m.selectFile != nil {
			return m.selectFile.View()
		}
		return ""
	case stateExportRunning, stateExportComplete:
		selected := "\n  Exporting to: " + m.styles.Selected.Render(m.filePath) + "\n"
		return selected + m.exportModel.View()
	}

	return ""
}

// StateExportAppConfig holds configuration for creating a new state export app.
type StateExportAppConfig struct {
	FilePath        string
	InstanceFilters []string
	EngineConfig    *stateio.EngineConfig
	Styles          *stylespkg.Styles
	Headless        bool
	HeadlessWriter  io.Writer
	JSONMode        bool
}

// NewStateExportApp creates a new state export application.
func NewStateExportApp(config StateExportAppConfig) (*MainModel, error) {
	// Determine if we're in auto-export mode (headless or file provided)
	autoExport := config.FilePath != "" || config.Headless

	// Determine the initial session state and create appropriate sub-models
	var sessionState stateExportSessionState
	var selectFile tea.Model

	if autoExport {
		// In headless/auto mode, go straight to running
		sessionState = stateExportRunning
	} else {
		// Interactive mode - start with file selection
		sessionState = stateExportFileSelect
		var err error
		selectFile, err = createSelectFileModel(config.Styles, "", false)
		if err != nil {
			return nil, err
		}
	}

	exportModel := NewExportModel(ExportModelConfig{
		EngineConfig:    config.EngineConfig,
		FilePath:        config.FilePath,
		InstanceFilters: config.InstanceFilters,
		Styles:          config.Styles,
		Headless:        config.Headless,
		HeadlessWriter:  config.HeadlessWriter,
		JSONMode:        config.JSONMode,
	})

	return &MainModel{
		sessionState: sessionState,
		filePath:     config.FilePath,
		selectFile:   selectFile,
		exportModel:  exportModel,
		styles:       config.Styles,
		width:        80,
	}, nil
}
