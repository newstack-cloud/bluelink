package sharedui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

type SelectWithPreviewModel struct {
	list          list.Model
	selected      string
	selectCommand func(string) tea.Cmd
	styles        *styles.BluelinkStyles
	width         int
	height        int
}

func (m SelectWithPreviewModel) Init() tea.Cmd {
	return nil
}

func (m SelectWithPreviewModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if m.list.FilterState() == list.Filtering {
			break
		}

		switch msg.String() {
		case "enter":
			item, ok := m.list.SelectedItem().(BluelinkListItem)
			if ok {
				m.selected = string(item.Key)
				cmds = append(cmds, m.selectCommand(m.selected))
			}
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		listWidth := msg.Width * 40 / 100
		m.list.SetWidth(listWidth)
		m.list.SetHeight(msg.Height - 4)
	}

	var listcmd tea.Cmd
	m.list, listcmd = m.list.Update(msg)
	cmds = append(cmds, listcmd)

	return m, tea.Batch(cmds...)
}

func (m SelectWithPreviewModel) View() string {
	selectedItem, ok := m.list.SelectedItem().(BluelinkListItem)
	var description string
	if ok {
		description = selectedItem.Desc
	}

	listWidth := m.width * 40 / 100
	previewWidth := m.width - listWidth - 4

	listStyle := lipgloss.NewStyle().
		Width(listWidth)

	previewStyle := lipgloss.NewStyle().
		Width(previewWidth).
		Padding(1, 2).
		MarginLeft(2).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(styles.BluelinkPrimary)

	previewTitleStyle := m.styles.Title.MarginBottom(1)

	previewContent := ""
	if selectedItem.Label != "" {
		previewContent = previewTitleStyle.Render(selectedItem.Label) + "\n\n"
	}
	if description != "" {
		descStyle := lipgloss.NewStyle().
			Width(previewWidth - 6).
			Foreground(styles.BluelinkMuted)
		previewContent += descStyle.Render(description)
	}

	listColumn := listStyle.Render(m.list.View())
	previewColumn := previewStyle.Render(previewContent)

	return lipgloss.JoinHorizontal(lipgloss.Top, listColumn, previewColumn)
}

func NewSelectWithPreview(
	title string,
	listItems []list.Item,
	bluelinkStyles *styles.BluelinkStyles,
	selectCommand func(string) tea.Cmd,
	enableFiltering bool,
) *SelectWithPreviewModel {
	const defaultWidth = 30
	const defaultHeight = 14

	sourceList := list.New(
		listItems,
		newItemDelegate(bluelinkStyles),
		defaultWidth,
		defaultHeight,
	)
	sourceList.Title = title
	sourceList.SetShowStatusBar(false)
	sourceList.SetFilteringEnabled(enableFiltering)
	sourceList.Styles.Title = bluelinkStyles.Title.MarginLeft(2)
	sourceList.Styles.PaginationStyle = bluelinkStyles.Pagination
	sourceList.Styles.HelpStyle = bluelinkStyles.Help

	return &SelectWithPreviewModel{
		list:          sourceList,
		selectCommand: selectCommand,
		styles:        bluelinkStyles,
		width:         80,
		height:        20,
	}
}
