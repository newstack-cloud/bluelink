package memfile

import (
	"time"

	"github.com/newstack-cloud/bluelink/libs/blueprint-state/manage"
)

func findIndexBeforeThreshold(
	entities []manage.Entity,
	thresholdDate time.Time,
) int {
	beforeThresholdIndex := -1
	i := len(entities) - 1

	for beforeThresholdIndex == -1 && i >= 0 {
		currentEntryTime := time.Unix(
			entities[i].GetCreated(),
			0,
		)
		if currentEntryTime.Before(thresholdDate) {
			beforeThresholdIndex = i
		}

		i -= 1
	}

	return beforeThresholdIndex
}
