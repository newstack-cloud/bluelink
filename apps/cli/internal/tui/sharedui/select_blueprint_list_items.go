package sharedui

import (
	"github.com/charmbracelet/bubbles/list"
	"github.com/newstack-cloud/bluelink/apps/cli/internal/consts"
)

func blueprintSourceListItems() []list.Item {
	return []list.Item{
		BluelinkListItem{
			Key:   consts.BlueprintSourceFile,
			Label: "Local file",
		},
		BluelinkListItem{
			Key:   consts.BlueprintSourceS3,
			Label: "AWS S3 Bucket",
		},
		BluelinkListItem{
			Key:   consts.BlueprintSourceGCS,
			Label: "Google Cloud Storage Bucket",
		},
		BluelinkListItem{
			Key:   consts.BlueprintSourceAzureBlob,
			Label: "Azure Blob Storage Container",
		},
		BluelinkListItem{
			Key:   consts.BlueprintSourceHTTPS,
			Label: "Public HTTPS URL",
		},
	}
}

func selectBlueprintStartListItems() []list.Item {
	return []list.Item{
		BluelinkListItem{
			Key:   "default",
			Label: "Use the default blueprint file",
		},
		BluelinkListItem{
			Key:   "select",
			Label: "Select a blueprint file",
		},
	}
}
