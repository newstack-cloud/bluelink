package sharedui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/styles"
)

// BluelinkListItem represents an item in a Bluelink list component.
type BluelinkListItem struct {
	Key   string
	Label string
	Desc  string
}

func (i BluelinkListItem) Title() string {
	return i.Label
}

func (i BluelinkListItem) Description() string {
	return i.Desc
}

func (i BluelinkListItem) FilterValue() string {
	return fmt.Sprintf("%s %s %s", i.Key, i.Label, i.Desc)
}

// ItemDelegate renders list items using Bluelink styles.
type ItemDelegate struct {
	styles *styles.BluelinkStyles
}

func newItemDelegate(bluelinkStyles *styles.BluelinkStyles) ItemDelegate {
	return ItemDelegate{
		styles: bluelinkStyles,
	}
}

func (d ItemDelegate) Height() int {
	return 1
}

func (d ItemDelegate) Spacing() int {
	return 0
}

func (d ItemDelegate) Update(_ tea.Msg, _ *list.Model) tea.Cmd {
	return nil
}

func (d ItemDelegate) Render(w io.Writer, m list.Model, index int, listItem list.Item) {
	i, ok := listItem.(BluelinkListItem)
	if !ok {
		return
	}

	if index == m.Index() {
		fmt.Fprint(w, d.styles.SelectedListItem.Render("> "+i.Label))
	} else {
		fmt.Fprint(w, d.styles.ListItem.Render(i.Label))
	}
}
