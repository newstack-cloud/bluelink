package helpersv1

import (
	"github.com/newstack-cloud/bluelink/apps/deploy-engine/internal/resolve"
)

func PopulateBlueprintDocInfoDefaults(payload *resolve.BlueprintDocumentInfo) {
	if payload == nil {
		return
	}

	if payload.FileSourceScheme == "" {
		payload.FileSourceScheme = DefaultFileSourceScheme
	}

	if payload.BlueprintFile == "" {
		payload.BlueprintFile = DefaultBlueprintFile
	}
}
