package provider

import (
	"time"

	"github.com/two-hundred/celerity/libs/blueprint/core"
)

// FieldChange represents a change in a field value
// of a resource or link that is used in change staging.
type FieldChange struct {
	FieldPath string            `json:"fieldPath"`
	PrevValue *core.MappingNode `json:"prevValue"`
	NewValue  *core.MappingNode `json:"newValue"`
	// MustRecreate is a flag that indicates whether the resource or link
	// containing the field must be recreated in order to apply the change.
	MustRecreate bool `json:"mustRecreate"`
}

// RetryContext contains information to be used for retrying operations
// such as resource deployment, data source fetching, etc.
type RetryContext struct {
	Attempt            int
	ExceededMaxRetries bool
	Policy             *RetryPolicy
	AttemptDurations   []float64
	AttemptStartTime   time.Time
}
