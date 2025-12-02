package initui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/tui/sharedui"
)

func selectTemplateListItems() []list.Item {
	return []list.Item{
		sharedui.BluelinkListItem{
			Key:   "scaffold",
			Label: "Scaffold",
			Desc:  "A scaffold project that generates essential files with placeholders.",
		},
		sharedui.BluelinkListItem{
			Key:   "aws-simple-api",
			Label: "AWS Simple API",
			Desc:  "A simple API project using AWS API Gateway and Lambda functions for a RESTful API.",
		},
	}
}
