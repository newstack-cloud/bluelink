package sharedui

import (
	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

type SelectModel struct {
	list          list.Model
	selected      string
	selectCommand func(string) tea.Cmd
}

func (m SelectModel) Init() tea.Cmd {
	return nil
}

func (m SelectModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	cmds := []tea.Cmd{}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Don't match any of the keys below if we're actively filtering.
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
		m.list.SetWidth(msg.Width)
	}

	var listcmd tea.Cmd
	m.list, listcmd = m.list.Update(msg)
	cmds = append(cmds, listcmd)

	return m, tea.Batch(cmds...)
}

func (m SelectModel) View() string {
	return m.list.View()
}

func NewSelect(
	title string,
	listItems []list.Item,
	bluelinkStyles *styles.BluelinkStyles,
	selectCommand func(string) tea.Cmd,
	enableFiltering bool,
) *SelectModel {
	const defaultWidth = 20
	sourceList := list.New(
		listItems,
		newItemDelegate(bluelinkStyles),
		defaultWidth,
		listHeight,
	)
	sourceList.Title = title
	sourceList.SetShowStatusBar(false)
	sourceList.SetFilteringEnabled(enableFiltering)
	sourceList.Styles.Title = bluelinkStyles.Title.MarginLeft(2)
	sourceList.Styles.PaginationStyle = bluelinkStyles.Pagination
	sourceList.Styles.HelpStyle = bluelinkStyles.Help

	return &SelectModel{
		list:          sourceList,
		selectCommand: selectCommand,
	}
}
