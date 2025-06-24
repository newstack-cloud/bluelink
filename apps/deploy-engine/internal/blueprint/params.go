package blueprint

import "github.com/newstack-cloud/bluelink/libs/blueprint/core"

// CreateEmptyBlueprintParams creates an empty BlueprintParams object
// with all fields initialized to empty maps or nil values.
func CreateEmptyBlueprintParams() core.BlueprintParams {
	return core.NewDefaultParams(
		map[string]map[string]*core.ScalarValue{},
		map[string]map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
		map[string]*core.ScalarValue{},
	)
}
