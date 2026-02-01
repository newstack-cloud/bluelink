package initui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/templates"
	sharedui "github.com/newstack-cloud/deploy-cli-sdk/ui"
)

func selectTemplateListItems() []list.Item {
	allTemplates := templates.GetTemplates()
	items := make([]list.Item, len(allTemplates))
	for i, t := range allTemplates {
		items[i] = sharedui.BluelinkListItem{
			Key:   t.Key,
			Label: t.Label,
			Desc:  t.Description,
		}
	}
	return items
}
