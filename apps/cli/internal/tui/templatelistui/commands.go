package templatelistui

import (
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/templates"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

// TemplatesLoadedMsg is sent when the template list has been loaded.
type TemplatesLoadedMsg struct {
	Templates []templates.Template
}

// TemplatesLoadErrorMsg is sent when loading the template list fails.
type TemplatesLoadErrorMsg struct {
	Err error
}

// TemplateSelectedMsg is sent when the user presses enter on a template.
type TemplateSelectedMsg struct {
	Key string
}

func selectTemplateCmd(key string) tea.Cmd {
	return func() tea.Msg {
		return TemplateSelectedMsg{Key: key}
	}
}

func loadTemplatesCmd(search string) tea.Cmd {
	return func() tea.Msg {
		allTemplates := templates.GetTemplates()
		filtered := filterBySearch(allTemplates, search)
		return TemplatesLoadedMsg{Templates: filtered}
	}
}

func filterBySearch(
	allTemplates []templates.Template,
	search string,
) []templates.Template {
	if search == "" {
		return allTemplates
	}

	lowerSearch := strings.ToLower(search)
	filtered := make([]templates.Template, 0, len(allTemplates))
	for _, t := range allTemplates {
		if strings.Contains(strings.ToLower(t.Key), lowerSearch) ||
			strings.Contains(strings.ToLower(t.Label), lowerSearch) ||
			strings.Contains(strings.ToLower(t.Description), lowerSearch) {
			filtered = append(filtered, t)
		}
	}
	return filtered
}

func templatesToListItems(tmpls []templates.Template) []list.Item {
	items := make([]list.Item, len(tmpls))
	for i, t := range tmpls {
		items[i] = sharedui.BluelinkListItem{
			Key:   t.Key,
			Label: t.Label,
			Desc:  t.Description,
		}
	}
	return items
}
