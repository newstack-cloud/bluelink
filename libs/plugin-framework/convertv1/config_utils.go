package convertv1

import (
	"fmt"
	"strings"

	"github.com/newstack-cloud/bluelink/libs/blueprint/core"
)

func expandNamespacedConfig(toExpand map[string]*core.ScalarValue) map[string]map[string]*core.ScalarValue {
	expanded := make(map[string]map[string]*core.ScalarValue)
	for k, v := range toExpand {
		ns, name := splitNamespacedKey(k)
		if _, ok := expanded[ns]; !ok {
			expanded[ns] = make(map[string]*core.ScalarValue)
		}
		// Ignore any keys that are not in the "{namespace}::{variableName}" format.
		if name != "" {
			expanded[ns][name] = v
		}
	}
	return expanded
}

func splitNamespacedKey(namespacedKey string) (string, string) {
	parts := strings.Split(namespacedKey, "::")
	if len(parts) == 1 {
		return parts[0], ""
	}
	return parts[0], parts[1]
}

func toNamespacedConfig(expanded map[string]map[string]*core.ScalarValue) map[string]*core.ScalarValue {
	toReturn := make(map[string]*core.ScalarValue)
	for ns, variables := range expanded {
		for name, value := range variables {
			namespacedKey := fmt.Sprintf("%s::%s", ns, name)
			toReturn[namespacedKey] = value
		}
	}
	return toReturn
}
