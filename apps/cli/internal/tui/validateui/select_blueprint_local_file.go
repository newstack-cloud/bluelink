package validateui

import (
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/filepicker"
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/consts"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

type SelectBlueprintMsg struct {
	blueprintFile string
}

type ClearSelectedBlueprintMsg struct{}

type SelectBlueprintModel struct {
	filepicker   filepicker.Model
	styles       styles.BluelinkStyles
	sourceList   list.Model
	source       string
	selectedFile string
	autoValidate bool
	stage        selectBlueprintStage
	quitting     bool
	err          error
}

type selectBlueprintStage int

const (
	// Stage where the user selects the source of the blueprint file.
	// Can be one of the following:
	// - "file" (local file)
	// - "https" (public URL)
	// - "s3" (AWS S3)
	// - "gcs" (Google Cloud Storage)
	// - "azureblob" (Azure Blob Storage)
	selectBlueprintStageSelectSource selectBlueprintStage = iota

	// Stage where the user inputs the location of the file
	// relative to a remote source scheme.
	// selectBlueprintStageInputFileLocation

	// Stage where the user selects a local file.
	selectBlueprintStageSelectLocalFile
)

const listHeight = 14

var (
	titleStyle      = lipgloss.NewStyle().MarginLeft(2)
	itemStyle       = lipgloss.NewStyle().PaddingLeft(4)
	paginationStyle = list.DefaultStyles().PaginationStyle.PaddingLeft(4)
	helpStyle       = list.DefaultStyles().HelpStyle.PaddingLeft(4).PaddingBottom(1)
)

type clearErrorMsg struct{}

func clearErrorAfter(t time.Duration) tea.Cmd {
	return tea.Tick(t, func(_ time.Time) tea.Msg {
		return clearErrorMsg{}
	})
}

func (m SelectBlueprintModel) Init() tea.Cmd {
	fcmd := m.filepicker.Init()
	if m.autoValidate {
		// Dispatch command to select the blueprint file
		// so the validation model can trigger the validation process.
		return tea.Batch(fcmd, selectBlueprintCmd(m.selectedFile))
	}
	return fcmd
}

func (m SelectBlueprintModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	prevStage := m.stage

	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			m.quitting = true
			return m, tea.Quit
		case "enter":
			if m.stage == selectBlueprintStageSelectSource {
				i, ok := m.sourceList.SelectedItem().(blueprintSourceItem)
				if ok {
					m.source = string(i.key)
					m.stage = selectBlueprintStageSelectLocalFile
				}
			}
		}
	case tea.WindowSizeMsg:
		var cmd tea.Cmd
		m.filepicker, cmd = m.filepicker.Update(msg)
		cmds = append(cmds, cmd)
	case clearErrorMsg:
		m.err = nil
	}

	var fpcmd tea.Cmd
	m.filepicker, fpcmd = m.filepicker.Update(msg)
	cmds = append(cmds, fpcmd)

	var listcmd tea.Cmd
	m.sourceList, listcmd = m.sourceList.Update(msg)
	cmds = append(cmds, listcmd)

	if prevStage != selectBlueprintStageSelectLocalFile {
		return m, tea.Batch(cmds...)
	}

	// Did the user select a file?
	if didSelect, path := m.filepicker.DidSelectFile(msg); didSelect {
		m.selectedFile = path
		// Dispatch comamand with the path of the selected file.
		cmds = append(cmds, selectBlueprintCmd(path))
	}

	// Did the user select a disabled file?
	// This is only necessary to display an error to the user.
	if didSelect, path := m.filepicker.DidSelectDisabledFile(msg); didSelect {
		// Let's clear the selectedFile and display an error.
		m.err = errors.New(path + " is not valid.")
		m.selectedFile = ""
		return m, tea.Batch(fpcmd, listcmd, clearErrorAfter(2*time.Second), clearSelectedBlueprintCmd())
	}

	return m, tea.Batch(cmds...)
}

func (m SelectBlueprintModel) View() string {
	if m.quitting {
		return ""
	}
	var s strings.Builder
	s.WriteString("\n  ")
	if m.err != nil {
		s.WriteString(m.filepicker.Styles.DisabledFile.Render(m.err.Error()))
	} else if m.selectedFile == "" {
		s.WriteString("Pick a blueprint file:")
	} else {
		s.WriteString("Blueprint file: " + m.styles.Selected.Render(m.selectedFile))
	}
	if m.stage == selectBlueprintStageSelectSource {
		s.WriteString("\n\n" + m.sourceList.View() + "\n")
	} else if m.stage == selectBlueprintStageSelectLocalFile {
		s.WriteString("\n\n" + m.filepicker.View() + "\n")
	}
	return s.String()
}

func NewSelectBlueprint(
	blueprintFile string,
	autoValidate bool,
	bluelinkStyles *styles.BluelinkStyles,
) (*SelectBlueprintModel, error) {
	fp := filepicker.New()
	fp.Styles = customFilePickerStyles(bluelinkStyles)
	fp.AllowedTypes = []string{".yaml", ".yml", ".json"}

	const defaultWidth = 20

	sourceListItems := blueprintSourceListItems()
	sourceList := list.New(sourceListItems, itemDelegate{}, defaultWidth, listHeight)
	sourceList.Title = "Where is the blueprint that you want to validate stored?"
	sourceList.SetShowStatusBar(false)
	sourceList.SetFilteringEnabled(false)
	sourceList.Styles.Title = titleStyle
	sourceList.Styles.PaginationStyle = paginationStyle
	sourceList.Styles.HelpStyle = helpStyle

	return &SelectBlueprintModel{
		filepicker:   fp,
		autoValidate: autoValidate,
		selectedFile: blueprintFile,
		sourceList:   sourceList,
		stage:        selectBlueprintStageSelectSource,
	}, nil
}

func customFilePickerStyles(bluelinkStyles *styles.BluelinkStyles) filepicker.Styles {
	styles := filepicker.DefaultStyles()
	styles.Selected = bluelinkStyles.Selected
	styles.File = bluelinkStyles.Selectable
	styles.Directory = bluelinkStyles.Selectable
	styles.Cursor = bluelinkStyles.Selected
	return styles
}

type blueprintSourceItem struct {
	key   string
	label string
}

func (i blueprintSourceItem) FilterValue() string {
	return ""
}

type itemDelegate struct{}

func (d itemDelegate) Height() int {
	return 1
}

func (d itemDelegate) Spacing() int {
	return 0
}

func (d itemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d itemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(blueprintSourceItem)
	if !ok {
		return
	}

	str := fmt.Sprintf("%d. %s", index+1, i.label)

	fn := itemStyle.Render
	if index == m.Index() {
		fn = func(s ...string) string {
			return selectedItemStyle.Render("> " + strings.Join(s, " "))
		}
	}

	fmt.Fprint(w, fn(str))
}

func blueprintSourceListItems() []list.Item {
	return []list.Item{
		blueprintSourceItem{
			key:   consts.BlueprintSourceFile,
			label: "Local file",
		},
		blueprintSourceItem{
			key:   consts.BlueprintSourceS3,
			label: "AWS S3 Bucket",
		},
		blueprintSourceItem{
			key:   consts.BlueprintSourceGCS,
			label: "Google Cloud Storage Bucket",
		},
		blueprintSourceItem{
			key:   consts.BlueprintSourceAzureBlob,
			label: "Azure Blob Storage Container",
		},
		blueprintSourceItem{
			key:   consts.BlueprintSourceHTTPS,
			label: "Public HTTPS URL",
		},
	}
}
